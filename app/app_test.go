// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	inventory_mocks "github.com/mendersoftware/deployments/client/inventory/mocks"
	reporting_mocks "github.com/mendersoftware/deployments/client/reporting/mocks"
	workflows_mocks "github.com/mendersoftware/deployments/client/workflows/mocks"
	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/identity"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string

		DataStoreError error
		FileStoreError error
		WorkflowsError error
		InventoryError error
		ReportingError error
	}{{
		Name: "ok",
	}, {
		Name:           "error: datastore",
		DataStoreError: errors.New("connection error"),
	}, {
		Name:           "error: filestore",
		FileStoreError: errors.New("connection error"),
	}, {
		Name:           "error: workflows",
		WorkflowsError: errors.New("connection error"),
	}, {
		Name:           "error: inventory",
		InventoryError: errors.New("connection error"),
	}, {
		Name:           "error: reporting",
		ReportingError: errors.New("connection error"),
	}}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()
			mDStore := &mocks.DataStore{}
			mFStore := &fs_mocks.FileStorage{}
			mWorkflows := &workflows_mocks.Client{}
			mInventory := &inventory_mocks.Client{}
			mReporting := &reporting_mocks.Client{}
			dep := &Deployments{
				db:              mDStore,
				fileStorage:     mFStore,
				workflowsClient: mWorkflows,
				inventoryClient: mInventory,
			}
			dep = dep.WithReporting(mReporting)
			switch {
			default:
				mReporting.On("CheckHealth", ctx).
					Return(tc.ReportingError)
				fallthrough
			case tc.InventoryError != nil:
				mInventory.On("CheckHealth", ctx).
					Return(tc.InventoryError)
				fallthrough
			case tc.WorkflowsError != nil:
				mWorkflows.On("CheckHealth", ctx).
					Return(tc.WorkflowsError)
				fallthrough
			case tc.FileStoreError != nil:
				mFStore.On("ListBuckets", ctx).
					Return(nil, tc.FileStoreError)
				fallthrough
			case tc.DataStoreError != nil:
				mDStore.On("Ping", ctx).
					Return(tc.DataStoreError)
				mDStore.On("GetStorageSettings", ctx).
					Return(&model.StorageSettings{
						Region:      config.Config.GetString(dconfig.SettingAwsS3Region),
						ExternalUri: config.Config.GetString(dconfig.SettingAwsExternalURI),
						Uri:         config.Config.GetString(dconfig.SettingAwsURI),
						Bucket:      config.Config.GetString(dconfig.SettingAwsS3Bucket),
						Key:         config.Config.GetString(dconfig.SettingAwsAuthKeyId),
						Secret:      config.Config.GetString(dconfig.SettingAwsAuthSecret),
						Token:       config.Config.GetString(dconfig.SettingAwsAuthToken)}, nil)
			}
			err := dep.HealthCheck(ctx)
			switch {
			case tc.DataStoreError != nil:
				assert.EqualError(t, err,
					"error reaching MongoDB: "+
						tc.DataStoreError.Error(),
				)

			case tc.FileStoreError != nil:
				assert.EqualError(t, err,
					"error reaching artifact storage service: "+
						tc.FileStoreError.Error(),
				)

			case tc.WorkflowsError != nil:
				assert.EqualError(t, err,
					"Workflows service unhealthy: "+
						tc.WorkflowsError.Error(),
				)

			case tc.InventoryError != nil:
				assert.EqualError(t, err,
					"Inventory service unhealthy: "+
						tc.InventoryError.Error(),
				)

			case tc.ReportingError != nil:
				assert.EqualError(t, err,
					"Reporting service unhealthy: "+
						tc.ReportingError.Error(),
				)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentModelCreateDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		InputConstructor *model.DeploymentConstructor

		InputDeploymentStorageInsertError error
		InputImagesByNameError            error

		InvDevices        []model.InvDevice
		InvDevicesPageTwo []model.InvDevice
		TotalCount        int
		SearchError       error
		GetFilterError    error

		CallGetDeviceGroups  bool
		InventoryGroups      []string
		GetDeviceGroupsError error

		ReportingService bool

		OutputError error
		OutputBody  bool
	}{
		"model missing": {
			OutputError: ErrModelMissingInput,
		},
		"insert error": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "NYC Production",
				ArtifactName: "App 123",
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeploymentStorageInsertError: errors.New("insert error"),
			CallGetDeviceGroups:               true,

			OutputError: errors.New("Storing deployment data: insert error"),
		},
		"ok": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "NYC Production",
				ArtifactName: "App 123",
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			CallGetDeviceGroups: true,
			InventoryGroups:     []string{"foo", "bar"},

			OutputBody: true,
		},
		"ok with group": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			InvDevices: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
				},
			},
			TotalCount: 1,

			OutputBody: true,
		},
		"ok with group, two pages": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			InvDevices: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
				},
			},
			InvDevicesPageTwo: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc348",
				},
			},
			TotalCount: 2,

			OutputBody: true,
		},
		"ok with group, reeporting": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			InvDevices: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
				},
			},
			TotalCount: 1,

			ReportingService: true,

			OutputBody: true,
		},
		"ko, with group, no device found": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			OutputError: ErrNoDevices,
		},
		"ko, with group, error while searching": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			SearchError: errors.New("error searching inventory"),
			OutputError: ErrModelInternal,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {
			ctx := context.Background()

			identityObject := &identity.Identity{Tenant: "tenant_id"}
			ctx = identity.WithContext(ctx, identityObject)

			db := mocks.DataStore{}
			db.On("InsertDeployment",
				ctx,
				mock.AnythingOfType("*model.Deployment")).
				Return(testCase.InputDeploymentStorageInsertError)

			db.On("ImagesByName",
				ctx,
				mock.AnythingOfType("string")).
				Return(
					[]*model.Image{model.NewImage(
						validUUIDv4,
						&model.ImageMeta{},
						&model.ArtifactMeta{
							Name: "App 123",
							DeviceTypesCompatible: []string{
								"hammer",
							},
							Depends: map[string]interface{}{},
						}, artifactSize)},
					testCase.InputImagesByNameError)

			fs := &fs_mocks.FileStorage{}
			ds := NewDeployments(&db, fs, "")

			mockInventoryClient := &inventory_mocks.Client{}
			if testCase.CallGetDeviceGroups {
				mockInventoryClient.On("GetDeviceGroups",
					ctx,
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string")).
					Return(testCase.InventoryGroups, testCase.GetDeviceGroupsError)
			}

			mockReportingClient := &reporting_mocks.Client{}
			if testCase.InputConstructor != nil && testCase.InputConstructor.Group != "" && len(testCase.InputConstructor.Devices) == 0 {
				if testCase.ReportingService {
					mockReportingClient.On("Search", ctx,
						"tenant_id",
						model.SearchParams{
							Page:    1,
							PerPage: PerPageInventoryDevices,
							Filters: []model.FilterPredicate{
								{
									Scope:     InventoryIdentityScope,
									Attribute: InventoryStatusAttributeName,
									Type:      "$eq",
									Value:     InventoryStatusAccepted,
								},
								{
									Scope:     InventoryGroupScope,
									Attribute: InventoryGroupAttributeName,
									Type:      "$eq",
									Value:     testCase.InputConstructor.Group,
								},
							},
						},
					).Return(testCase.InvDevices, testCase.TotalCount, testCase.SearchError)
				} else {
					mockInventoryClient.On("Search", ctx,
						"tenant_id",
						model.SearchParams{
							Page:    1,
							PerPage: PerPageInventoryDevices,
							Filters: []model.FilterPredicate{
								{
									Scope:     InventoryIdentityScope,
									Attribute: InventoryStatusAttributeName,
									Type:      "$eq",
									Value:     InventoryStatusAccepted,
								},
								{
									Scope:     InventoryGroupScope,
									Attribute: InventoryGroupAttributeName,
									Type:      "$eq",
									Value:     testCase.InputConstructor.Group,
								},
							},
						},
					).Return(testCase.InvDevices, testCase.TotalCount, testCase.SearchError)

					if testCase.TotalCount > len(testCase.InvDevices) {
						mockInventoryClient.On("Search", ctx,
							"tenant_id",
							model.SearchParams{
								Page:    2,
								PerPage: PerPageInventoryDevices,
								Filters: []model.FilterPredicate{
									{
										Scope:     InventoryIdentityScope,
										Attribute: InventoryStatusAttributeName,
										Type:      "$eq",
										Value:     InventoryStatusAccepted,
									},
									{
										Scope:     InventoryGroupScope,
										Attribute: InventoryGroupAttributeName,
										Type:      "$eq",
										Value:     testCase.InputConstructor.Group,
									},
								},
							},
						).Return(testCase.InvDevicesPageTwo, testCase.TotalCount, testCase.SearchError)
					}
				}
			}

			ds.SetInventoryClient(mockInventoryClient)
			if testCase.ReportingService {
				ds.WithReporting(mockReportingClient)
			}

			out, err := ds.CreateDeployment(ctx, testCase.InputConstructor)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			if testCase.OutputBody {
				assert.NotNil(t, out)
			}

			mockInventoryClient.AssertExpectations(t)
		})
	}

}

func TestCreateDeviceConfigurationDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		inputConstructor  *model.ConfigurationDeploymentConstructor
		inputDeviceID     string
		inputDeploymentID string

		inputDeploymentStorageInsertError error
		inventoryError                    error

		callInventory bool
		callDb        bool

		outputError error
		outputID    string
	}{
		"ok": {
			inputConstructor: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: []byte("bar"),
			},
			inputDeviceID:     "foo-device",
			inputDeploymentID: "foo-deployment",
			callInventory:     true,
			callDb:            true,

			outputID: "foo-deployment",
		},
		"constructor missing": {
			outputError: ErrModelMissingInput,
		},
		"insert error": {
			inputConstructor: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: []byte("bar"),
			},
			inputDeploymentStorageInsertError: errors.New("insert error"),
			callInventory:                     true,
			callDb:                            true,

			outputError: errors.New("Storing deployment data: insert error"),
		},
		"inventory error": {
			inputConstructor: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: []byte("bar"),
			},
			inventoryError: errors.New("inventory error"),
			callInventory:  true,

			outputError: errors.New("inventory error"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", name), func(t *testing.T) {
			ctx := context.Background()

			identityObject := &identity.Identity{Tenant: "tenant_id"}
			ctx = identity.WithContext(ctx, identityObject)

			db := mocks.DataStore{}
			if tc.callDb {
				db.On("InsertDeployment",
					ctx,
					mock.AnythingOfType("*model.Deployment")).
					Return(tc.inputDeploymentStorageInsertError)
			}
			defer db.AssertExpectations(t)

			inv := &inventory_mocks.Client{}
			if tc.callInventory {
				inv.On("GetDeviceGroups", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return([]string{}, tc.inventoryError)
			}
			defer inv.AssertExpectations(t)

			ds := &Deployments{
				db:              &db,
				inventoryClient: inv,
			}

			out, err := ds.CreateDeviceConfigurationDeployment(ctx, tc.inputConstructor, tc.inputDeviceID, tc.inputDeploymentID)
			if tc.outputError != nil {
				assert.EqualError(t, err, tc.outputError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, out, tc.outputID)
			}
		})
	}
}

func TestAbortDeployment(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		InputDeploymentID string

		AbortDeviceDeploymentsError error

		AggregateDeviceDeploymentByStatusStats model.Stats
		AggregateDeviceDeploymentByStatusError error
		CallAggregateDeviceDeploymentByStatus  bool

		UpdateStatsError error
		CallUpdateStats  bool

		SetDeploymentStatusError error
		CallSetDeploymentStatus  bool

		OutputError error
	}{
		"AbortDeviceDeployments error": {
			InputDeploymentID:           "f826484e-1157-4109-af21-304e6d711561",
			AbortDeviceDeploymentsError: errors.New("AbortDeviceDeploymentsError"),
			OutputError:                 errors.New("AbortDeviceDeploymentsError"),
		},
		"AggregateDeviceDeploymentByStatus error": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			CallAggregateDeviceDeploymentByStatus:  true,
			AggregateDeviceDeploymentByStatusError: errors.New("AggregateDeviceDeploymentByStatusError"),
			AggregateDeviceDeploymentByStatusStats: model.Stats{},
			OutputError:                            errors.New("AggregateDeviceDeploymentByStatusError"),
		},
		"UpdateStats error": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			CallAggregateDeviceDeploymentByStatus:  true,
			AggregateDeviceDeploymentByStatusStats: model.Stats{"aaa": 1},
			CallUpdateStats:                        true,
			UpdateStatsError:                       errors.New("UpdateStatsError"),
			OutputError:                            errors.New("failed to update deployment stats: UpdateStatsError"),
		},
		"all correct": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			CallAggregateDeviceDeploymentByStatus:  true,
			AggregateDeviceDeploymentByStatusStats: model.Stats{"aaa": 1},
			CallUpdateStats:                        true,
			CallSetDeploymentStatus:                true,
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", name), func(t *testing.T) {
			db := mocks.DataStore{}
			defer db.AssertExpectations(t)
			db.On("AbortDeviceDeployments",
				h.ContextMatcher(), tc.InputDeploymentID).
				Return(tc.AbortDeviceDeploymentsError)
			if tc.CallAggregateDeviceDeploymentByStatus {
				db.On("AggregateDeviceDeploymentByStatus",
					h.ContextMatcher(), tc.InputDeploymentID).
					Return(tc.AggregateDeviceDeploymentByStatusStats,
						tc.AggregateDeviceDeploymentByStatusError)
			}
			if tc.CallUpdateStats {
				db.On("UpdateStats",
					h.ContextMatcher(), tc.InputDeploymentID,
					mock.AnythingOfType("model.Stats")).
					Return(tc.UpdateStatsError)
			}
			if tc.CallSetDeploymentStatus {
				db.On("SetDeploymentStatus",
					h.ContextMatcher(), tc.InputDeploymentID,
					model.DeploymentStatusFinished, mock.AnythingOfType("time.Time")).
					Return(tc.SetDeploymentStatusError)
			}

			ds := &Deployments{
				db: &db,
			}
			ctx := context.Background()

			err := ds.AbortDeployment(ctx, tc.InputDeploymentID)
			if tc.OutputError != nil {
				assert.EqualError(t, err, tc.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestImageUsedInActiveDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		InputID string

		ExistUnfinishedByArtifactIdResponse bool
		ExistUnfinishedByArtifactIdError    error

		CallExistAssignedImageWithIDAndStatuses     bool
		ExistAssignedImageWithIDAndStatusesResponse bool
		ExistAssignedImageWithIDAndStatusesError    error

		OutputError error
		OutputBool  bool
	}{
		"ok": {
			InputID: "ID:1234",
			ExistAssignedImageWithIDAndStatusesResponse: true,
			CallExistAssignedImageWithIDAndStatuses:     true,

			OutputBool: true,
		},
		"ExistAssignedImageWithIDAndStatuses error": {
			InputID:                                  "ID:1234",
			ExistAssignedImageWithIDAndStatusesError: errors.New("Some error"),
			CallExistAssignedImageWithIDAndStatuses:  true,

			OutputError: errors.New("Checking if image is used by active deployment: Some error"),
		},
		"ExistUnfinishedByArtifactId error": {
			InputID:                             "ID:1234",
			ExistUnfinishedByArtifactIdError:    errors.New("Some error"),
			ExistUnfinishedByArtifactIdResponse: false,

			OutputError: errors.New("Checking if image is used by active deployment: Some error"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", name), func(t *testing.T) {
			db := mocks.DataStore{}
			defer db.AssertExpectations(t)

			db.On("ExistUnfinishedByArtifactId",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(tc.ExistUnfinishedByArtifactIdResponse,
					tc.ExistUnfinishedByArtifactIdError)

			if tc.CallExistAssignedImageWithIDAndStatuses {
				call := db.On("ExistAssignedImageWithIDAndStatuses",
					h.ContextMatcher(),
					tc.InputID).
					Return(tc.ExistAssignedImageWithIDAndStatusesResponse,
						tc.ExistAssignedImageWithIDAndStatusesError)
				varArgs := model.ActiveDeploymentStatuses()
				for i := range varArgs {
					call.Arguments = append(call.Arguments, varArgs[i])
				}
			}

			ds := &Deployments{
				db: &db,
			}

			found, err := ds.ImageUsedInActiveDeployment(context.Background(),
				tc.InputID)
			if tc.OutputError != nil {
				assert.EqualError(t, err, tc.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.OutputBool, found)
		})
	}

}
