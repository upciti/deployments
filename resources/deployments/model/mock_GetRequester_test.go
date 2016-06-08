// Copyright 2016 Mender Software AS
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

package model

import (
	"time"

	"github.com/mendersoftware/deployments/resources/images"
	"github.com/stretchr/testify/mock"
)

// MockGetRequester is an autogenerated mock type for the GetRequester type
type MockGetRequester struct {
	mock.Mock
}

// GetRequest provides a mock function with given fields: objectId, duration
func (_m *MockGetRequester) GetRequest(objectId string, duration time.Duration) (*images.Link, error) {
	ret := _m.Called(objectId, duration)

	var r0 *images.Link
	if rf, ok := ret.Get(0).(func(string, time.Duration) *images.Link); ok {
		r0 = rf(objectId, duration)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*images.Link)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, time.Duration) error); ok {
		r1 = rf(objectId, duration)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
