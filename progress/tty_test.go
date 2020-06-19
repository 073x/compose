/*
   Copyright 2020 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package progress

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLineText(t *testing.T) {
	now := time.Now()
	ev := Event{
		ID:         "id",
		Text:       "Text",
		Status:     Working,
		StatusText: "Status",
		endTime:    now,
		startTime:  now,
		spinner: &spinner{
			chars: []string{"."},
		},
	}

	lineWidth := len(fmt.Sprintf("%s %s", ev.ID, ev.Text))

	out := lineText(ev, 50, lineWidth, true)
	assert.Equal(t, "\x1b[37m . id Text Status                            0.0s\n\x1b[0m", out)

	out = lineText(ev, 50, lineWidth, false)
	assert.Equal(t, " . id Text Status                            0.0s\n", out)

	ev.Status = Done
	out = lineText(ev, 50, lineWidth, true)
	assert.Equal(t, "\x1b[34m . id Text Status                            0.0s\n\x1b[0m", out)

	ev.Status = Error
	out = lineText(ev, 50, lineWidth, true)
	assert.Equal(t, "\x1b[31m . id Text Status                            0.0s\n\x1b[0m", out)
}
