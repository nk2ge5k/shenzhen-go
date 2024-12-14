//go:build webview
// +build webview

// Copyright 2018 Google Inc.
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

package main

import (
	"flag"

	webview "github.com/webview/webview_go"
)

func init() {
	useDefaultBrowser = flag.Bool("use_browser", false, "Load in the system's default web browser instead of the inbuilt webview")
	webviewOpen = func(url string) error {
		return webview.Open("Shenzhen Go", url, 1152, 720, true)
	}
}
