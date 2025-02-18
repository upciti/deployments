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

package testing

import (
	"io/ioutil"
	"os"
)

func CreateValidImageFile() *os.File {
	someData := []byte{115, 111, 109, 101, 10, 11}
	tmpfile, _ := ioutil.TempFile("", "artifact-")
	_, _ = tmpfile.Write(someData)
	_, _ = tmpfile.Seek(0, 0)
	return tmpfile
}
