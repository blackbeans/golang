// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imports

import "io"

import (
	a "io"
)

import a "io"

import (
	"io";
	"io";
	"io";
)

import (
	"io";
	aLongRename "io";
	b "io";
	c "i" "o";
)

// TODO(gri) add more test cases
