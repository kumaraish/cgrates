/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package ers

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/xmlquery"

	"github.com/cgrates/cgrates/agents"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func NewXMLFileER(cfg *config.CGRConfig, cfgIdx int,
	rdrEvents, partialEvents chan *erEvent, rdrErr chan error,
	fltrS *engine.FilterS, rdrExit chan struct{}) (er EventReader, err error) {
	srcPath := cfg.ERsCfg().Readers[cfgIdx].SourcePath
	if strings.HasSuffix(srcPath, utils.Slash) {
		srcPath = srcPath[:len(srcPath)-1]
	}
	xmlER := &XMLFileER{
		cgrCfg:        cfg,
		cfgIdx:        cfgIdx,
		fltrS:         fltrS,
		rdrDir:        srcPath,
		rdrEvents:     rdrEvents,
		partialEvents: partialEvents,
		rdrError:      rdrErr,
		rdrExit:       rdrExit,
		conReqs:       make(chan struct{}, cfg.ERsCfg().Readers[cfgIdx].ConcurrentReqs)}
	var processFile struct{}
	for i := 0; i < cfg.ERsCfg().Readers[cfgIdx].ConcurrentReqs; i++ {
		xmlER.conReqs <- processFile // Empty initiate so we do not need to wait later when we pop
	}
	return xmlER, nil
}

// XMLFileER implements EventReader interface for .xml files
type XMLFileER struct {
	sync.RWMutex
	cgrCfg        *config.CGRConfig
	cfgIdx        int // index of config instance within ERsCfg.Readers
	fltrS         *engine.FilterS
	rdrDir        string
	rdrEvents     chan *erEvent // channel to dispatch the events created to
	partialEvents chan *erEvent // channel to dispatch the partial events created to
	rdrError      chan error
	rdrExit       chan struct{}
	conReqs       chan struct{} // limit number of opened files
}

func (rdr *XMLFileER) Config() *config.EventReaderCfg {
	return rdr.cgrCfg.ERsCfg().Readers[rdr.cfgIdx]
}

func (rdr *XMLFileER) Serve() (err error) {
	switch rdr.Config().RunDelay {
	case time.Duration(0): // 0 disables the automatic read, maybe done per API
		return
	case time.Duration(-1):
		return utils.WatchDir(rdr.rdrDir, rdr.processFile,
			utils.ERs, rdr.rdrExit)
	default:
		go func() {
			tm := time.NewTimer(0)
			for {
				// Not automated, process and sleep approach
				select {
				case <-rdr.rdrExit:
					tm.Stop()
					utils.Logger.Info(
						fmt.Sprintf("<%s> stop monitoring path <%s>",
							utils.ERs, rdr.rdrDir))
					return
				case <-tm.C:
				}
				filesInDir, _ := os.ReadDir(rdr.rdrDir)
				for _, file := range filesInDir {
					if !strings.HasSuffix(file.Name(), utils.XMLSuffix) { // hardcoded file extension for xml event reader
						continue // used in order to filter the files from directory
					}
					go func(fileName string) {
						if err := rdr.processFile(rdr.rdrDir, fileName); err != nil {
							utils.Logger.Warning(
								fmt.Sprintf("<%s> processing file %s, error: %s",
									utils.ERs, fileName, err.Error()))
						}
					}(file.Name())
				}
				tm.Reset(rdr.Config().RunDelay)
			}
		}()
	}
	return
}

// processFile is called for each file in a directory and dispatches erEvents from it
func (rdr *XMLFileER) processFile(fPath, fName string) (err error) {
	if cap(rdr.conReqs) != 0 { // 0 goes for no limit
		processFile := <-rdr.conReqs // Queue here for maxOpenFiles
		defer func() { rdr.conReqs <- processFile }()
	}
	absPath := path.Join(fPath, fName)
	utils.Logger.Info(
		fmt.Sprintf("<%s> parsing <%s>", utils.ERs, absPath))
	var file *os.File
	if file, err = os.Open(absPath); err != nil {
		return
	}
	defer file.Close()
	doc, err := xmlquery.Parse(file)
	if err != nil {
		return err
	}
	var xmlRootPath utils.HierarchyPath
	if rdr.Config().Opts.XMLRootPath != nil {
		xmlRootPath = utils.ParseHierarchyPath(*rdr.Config().Opts.XMLRootPath, utils.EmptyString)
	}
	xmlElmts := xmlquery.Find(doc, xmlRootPath.AsString("/", true))
	rowNr := 0 // This counts the rows in the file, not really number of CDRs
	evsPosted := 0
	timeStart := time.Now()
	reqVars := &utils.DataNode{Type: utils.NMMapType, Map: map[string]*utils.DataNode{utils.FileName: utils.NewLeafNode(fName)}}
	for _, xmlElmt := range xmlElmts {
		rowNr++ // increment the rowNr after checking if it's not the end of file
		agReq := agents.NewAgentRequest(
			config.NewXMLProvider(xmlElmt, xmlRootPath), reqVars,
			nil, nil, nil, rdr.Config().Tenant,
			rdr.cgrCfg.GeneralCfg().DefaultTenant,
			utils.FirstNonEmpty(rdr.Config().Timezone,
				rdr.cgrCfg.GeneralCfg().DefaultTimezone),
			rdr.fltrS, nil) // create an AgentRequest
		if pass, err := rdr.fltrS.Pass(agReq.Tenant, rdr.Config().Filters,
			agReq); err != nil {
			utils.Logger.Warning(
				fmt.Sprintf("<%s> reading file: <%s> row <%d>, ignoring due to filter error: <%s>",
					utils.ERs, absPath, rowNr, err.Error()))
			return err
		} else if !pass {
			continue
		}
		if err = agReq.SetFields(rdr.Config().Fields); err != nil {
			utils.Logger.Warning(
				fmt.Sprintf("<%s> reading file: <%s> row <%d>, ignoring due to error: <%s>",
					utils.ERs, absPath, rowNr, err.Error()))
			continue
		}
		cgrEv := utils.NMAsCGREvent(agReq.CGRRequest, agReq.Tenant, utils.NestingSep, agReq.Opts)
		rdrEv := rdr.rdrEvents
		if _, isPartial := cgrEv.APIOpts[utils.PartialOpt]; isPartial {
			rdrEv = rdr.partialEvents
		}
		rdrEv <- &erEvent{
			cgrEvent: cgrEv,
			rdrCfg:   rdr.Config(),
		}
		evsPosted++
	}

	if rdr.Config().ProcessedPath != "" {
		// Finished with file, move it to processed folder
		outPath := path.Join(rdr.Config().ProcessedPath, fName)
		if err = os.Rename(absPath, outPath); err != nil {
			return
		}
	}

	utils.Logger.Info(
		fmt.Sprintf("%s finished processing file <%s>. Total records processed: %d, events posted: %d, run duration: %s",
			utils.ERs, absPath, rowNr, evsPosted, time.Since(timeStart)))
	return
}
