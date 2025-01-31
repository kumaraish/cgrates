//go:build offline
// +build offline

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

package v1

import (
	"net/rpc"
	"net/rpc/jsonrpc"
	"path"
	"reflect"
	"testing"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

var (
	tpActionTriggerCfgPath   string
	tpActionTriggerCfg       *config.CGRConfig
	tpActionTriggerRPC       *rpc.Client
	tpActionTriggers         *utils.TPActionTriggers
	tpActionTriggerDelay     int
	tpActionTriggerConfigDIR string //run tests for specific configuration

)

var sTestsTPActionTriggers = []func(t *testing.T){
	testTPActionTriggersInitCfg,
	testTPActionTriggersResetStorDb,
	testTPActionTriggersStartEngine,
	testTPActionTriggersRpcConn,
	testTPActionTriggersGetTPActionTriggersBeforeSet,
	testTPActionTriggersSetTPActionTriggers,
	testTPActionTriggersGetTPActionTriggersAfterSet,
	testTPActionTriggersGetTPActionTriggersIds,
	testTPActionTriggersUpdateTPActionTriggers,
	testTPActionTriggersGetTPActionTriggersAfterUpdate,
	testTPActionTriggersRemoveTPActionTriggers,
	testTPActionTriggersGetTPActionTriggersAfterRemove,
	testTPActionTriggersKillEngine,
}

//Test start here
func TestTPActionTriggersIT(t *testing.T) {
	switch *dbType {
	case utils.MetaInternal:
		tpActionTriggerConfigDIR = "tutinternal"
	case utils.MetaMySQL:
		tpActionTriggerConfigDIR = "tutmysql"
	case utils.MetaMongo:
		tpActionTriggerConfigDIR = "tutmongo"
	case utils.MetaPostgres:
		tpActionTriggerConfigDIR = "tutpostgres"
	default:
		t.Fatal("Unknown Database type")
	}
	for _, stest := range sTestsTPActionTriggers {
		t.Run(tpActionTriggerConfigDIR, stest)
	}
}

func testTPActionTriggersInitCfg(t *testing.T) {
	var err error
	tpActionTriggerCfgPath = path.Join(*dataDir, "conf", "samples", tpActionTriggerConfigDIR)
	tpActionTriggerCfg, err = config.NewCGRConfigFromPath(tpActionTriggerCfgPath)
	if err != nil {
		t.Error(err)
	}
	switch tpActionTriggerConfigDIR {
	case "tutmongo": // Mongo needs more time to reset db, need to investigate
		tpActionTriggerDelay = 2000
	default:
		tpActionTriggerDelay = 1000
	}
}

// Wipe out the cdr database
func testTPActionTriggersResetStorDb(t *testing.T) {
	if err := engine.InitStorDb(tpActionTriggerCfg); err != nil {
		t.Fatal(err)
	}
}

// Start CGR Engine
func testTPActionTriggersStartEngine(t *testing.T) {
	if _, err := engine.StopStartEngine(tpActionTriggerCfgPath, tpActionTriggerDelay); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func testTPActionTriggersRpcConn(t *testing.T) {
	var err error
	tpActionTriggerRPC, err = jsonrpc.Dial(utils.TCP, tpActionTriggerCfg.ListenCfg().RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal(err)
	}
}

func testTPActionTriggersGetTPActionTriggersBeforeSet(t *testing.T) {
	var reply *utils.TPActionTriggers
	if err := tpActionTriggerRPC.Call(utils.APIerSv1GetTPActionTriggers,
		&AttrGetTPActionTriggers{TPid: "TPAct", ID: "ID"}, &reply); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
}

func testTPActionTriggersSetTPActionTriggers(t *testing.T) {
	tpActionTriggers = &utils.TPActionTriggers{
		TPid: "TPAct",
		ID:   "ID",
		ActionTriggers: []*utils.TPActionTrigger{
			&utils.TPActionTrigger{
				Id:                    "STANDARD_TRIGGERS",
				UniqueID:              "",
				ThresholdType:         "*min_balance",
				ThresholdValue:        2,
				Recurrent:             false,
				MinSleep:              "0",
				ExpirationDate:        "",
				ActivationDate:        "",
				BalanceId:             "",
				BalanceType:           "*monetary",
				BalanceDestinationIds: "FS_USERS",
				BalanceWeight:         "",
				BalanceExpirationDate: "",
				BalanceTimingTags:     "",
				BalanceRatingSubject:  "",
				BalanceCategories:     "",
				BalanceSharedGroups:   "",
				BalanceBlocker:        "",
				BalanceDisabled:       "",
				ActionsId:             "LOG_WARNING",
				Weight:                10,
			},
			&utils.TPActionTrigger{
				Id:                    "STANDARD_TRIGGERS",
				UniqueID:              "",
				ThresholdType:         "*max_event_counter",
				ThresholdValue:        5,
				Recurrent:             false,
				MinSleep:              "0",
				ExpirationDate:        "",
				ActivationDate:        "",
				BalanceId:             "",
				BalanceType:           "*monetary",
				BalanceDestinationIds: "FS_USERS",
				BalanceWeight:         "",
				BalanceExpirationDate: "",
				BalanceTimingTags:     "",
				BalanceRatingSubject:  "",
				BalanceCategories:     "",
				BalanceSharedGroups:   "",
				BalanceBlocker:        "",
				BalanceDisabled:       "",
				ActionsId:             "LOG_WARNING",
				Weight:                10,
			},
		},
	}
	var result string
	if err := tpActionTriggerRPC.Call(utils.APIerSv1SetTPActionTriggers, tpActionTriggers, &result); err != nil {
		t.Error(err)
	} else if result != utils.OK {
		t.Error("Unexpected reply returned", result)
	}
}

func testTPActionTriggersGetTPActionTriggersAfterSet(t *testing.T) {
	var reply *utils.TPActionTriggers
	if err := tpActionTriggerRPC.Call(utils.APIerSv1GetTPActionTriggers,
		&AttrGetTPActionTriggers{TPid: "TPAct", ID: "ID"}, &reply); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(tpActionTriggers.TPid, reply.TPid) {
		t.Errorf("Expecting : %+v, received: %+v", tpActionTriggers.TPid, reply.TPid)
	} else if !reflect.DeepEqual(tpActionTriggers.ID, reply.ID) {
		t.Errorf("Expecting : %+v, received: %+v", tpActionTriggers.ID, reply.ID)
	} else if !reflect.DeepEqual(len(tpActionTriggers.ActionTriggers), len(reply.ActionTriggers)) {
		t.Errorf("Expecting : %+v, received: %+v", len(tpActionTriggers.ActionTriggers), len(reply.ActionTriggers))
	}
}

func testTPActionTriggersGetTPActionTriggersIds(t *testing.T) {
	var result []string
	expectedTPID := []string{"ID"}
	if err := tpActionTriggerRPC.Call(utils.APIerSv1GetTPActionTriggerIds,
		&AttrGetTPActionTriggerIds{TPid: "TPAct"}, &result); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(expectedTPID, result) {
		t.Errorf("Expecting: %+v, received: %+v", expectedTPID, result)
	}
}

func testTPActionTriggersUpdateTPActionTriggers(t *testing.T) {
	tpActionTriggers.ActionTriggers = []*utils.TPActionTrigger{
		&utils.TPActionTrigger{
			Id:                    "STANDARD_TRIGGERS",
			UniqueID:              "",
			ThresholdType:         "*min_balance",
			ThresholdValue:        2,
			Recurrent:             false,
			MinSleep:              "0",
			ExpirationDate:        "",
			ActivationDate:        "",
			BalanceId:             "",
			BalanceType:           "*monetary",
			BalanceDestinationIds: "FS_USERS",
			BalanceWeight:         "",
			BalanceExpirationDate: "",
			BalanceTimingTags:     "",
			BalanceRatingSubject:  "",
			BalanceCategories:     "",
			BalanceSharedGroups:   "",
			BalanceBlocker:        "",
			BalanceDisabled:       "",
			ActionsId:             "LOG_WARNING",
			Weight:                10,
		},
		&utils.TPActionTrigger{
			Id:                    "STANDARD_TRIGGERS",
			UniqueID:              "",
			ThresholdType:         "*max_event_counter",
			ThresholdValue:        5,
			Recurrent:             false,
			MinSleep:              "0",
			ExpirationDate:        "",
			ActivationDate:        "",
			BalanceId:             "",
			BalanceType:           "*monetary",
			BalanceDestinationIds: "FS_USERS",
			BalanceWeight:         "",
			BalanceExpirationDate: "",
			BalanceTimingTags:     "",
			BalanceRatingSubject:  "",
			BalanceCategories:     "",
			BalanceSharedGroups:   "",
			BalanceBlocker:        "",
			BalanceDisabled:       "",
			ActionsId:             "LOG_WARNING",
			Weight:                10,
		},
		&utils.TPActionTrigger{
			Id:                    "CDRST1_WARN",
			UniqueID:              "",
			ThresholdType:         "*min_asr",
			ThresholdValue:        45,
			Recurrent:             true,
			MinSleep:              "1m",
			ExpirationDate:        "",
			ActivationDate:        "",
			BalanceId:             "",
			BalanceType:           "",
			BalanceDestinationIds: "",
			BalanceWeight:         "",
			BalanceExpirationDate: "",
			BalanceTimingTags:     "",
			BalanceRatingSubject:  "",
			BalanceCategories:     "",
			BalanceSharedGroups:   "",
			BalanceBlocker:        "",
			BalanceDisabled:       "",
			ActionsId:             "LOG_WARNING",
			Weight:                10,
		},
	}

	var result string
	if err := tpActionTriggerRPC.Call(utils.APIerSv1SetTPActionTriggers, tpActionTriggers, &result); err != nil {
		t.Error(err)
	} else if result != utils.OK {
		t.Error("Unexpected reply returned", result)
	}

}

func testTPActionTriggersGetTPActionTriggersAfterUpdate(t *testing.T) {
	var reply *utils.TPActionTriggers
	if err := tpActionTriggerRPC.Call(utils.APIerSv1GetTPActionTriggers,
		&AttrGetTPActionTriggers{TPid: "TPAct", ID: "ID"}, &reply); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(tpActionTriggers.TPid, reply.TPid) {
		t.Errorf("Expecting : %+v, received: %+v", tpActionTriggers.TPid, reply.TPid)
	} else if !reflect.DeepEqual(tpActionTriggers.ID, reply.ID) {
		t.Errorf("Expecting : %+v, received: %+v", tpActionTriggers.ID, reply.ID)
	} else if !reflect.DeepEqual(len(tpActionTriggers.ActionTriggers), len(reply.ActionTriggers)) {
		t.Errorf("Expecting : %+v, received: %+v", len(tpActionTriggers.ActionTriggers), len(reply.ActionTriggers))
	}

}

func testTPActionTriggersRemoveTPActionTriggers(t *testing.T) {
	var resp string
	if err := tpActionTriggerRPC.Call(utils.APIerSv1RemoveTPActionTriggers,
		&AttrGetTPActionTriggers{TPid: "TPAct", ID: "ID"}, &resp); err != nil {
		t.Error(err)
	} else if resp != utils.OK {
		t.Error("Unexpected reply returned", resp)
	}

}

func testTPActionTriggersGetTPActionTriggersAfterRemove(t *testing.T) {
	var reply *utils.TPActionTriggers
	if err := tpActionTriggerRPC.Call(utils.APIerSv1GetTPActionTriggers,
		&AttrGetTPActionTriggers{TPid: "TPAct", ID: "ID"}, &reply); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
}

func testTPActionTriggersKillEngine(t *testing.T) {
	if err := engine.KillEngine(tpActionTriggerDelay); err != nil {
		t.Error(err)
	}
}
