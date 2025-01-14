// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (self *SGuest) UpdateQgaStatus(status string) error {
	_, err := db.Update(self, func() error {
		self.QgaStatus = status
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update QgaStatus")
	}
	return nil
}

func (self *SGuest) PerformQgaSetPassword(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaSetPasswordInput,
) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_RUNNING {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	if input.Username == "" {
		return nil, httperrors.NewMissingParameterError("username")
	}
	if input.Password == "" {
		return nil, httperrors.NewMissingParameterError("password")
	}
	err := seclib2.ValidatePassword(input.Password)
	if err != nil {
		return nil, err
	}
	self.SetStatus(userCred, api.VM_QGA_SET_PASSWORD, "")
	self.UpdateQgaStatus(api.QGA_STATUS_EXCUTING)
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "GuestQgaSetPasswordTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SGuest) PerformQgaCommand(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaCommandInput,
) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_RUNNING {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	if input.Command == "" {
		return nil, httperrors.NewMissingParameterError("command")
	}
	host, _ := self.GetHost()
	self.SetStatus(userCred, api.VM_QGA_COMMAND_EXECUTING, "qga command")
	self.UpdateQgaStatus(api.QGA_STATUS_EXCUTING)
	defer self.SetStatus(userCred, api.VM_RUNNING, "qga comm")
	defer self.UpdateQgaStatus(api.QGA_STATUS_AVAILABLE)

	return self.GetDriver().RequestQgaCommand(ctx, userCred, jsonutils.Marshal(input), host, self)
}
