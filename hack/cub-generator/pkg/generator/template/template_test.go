// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package template_test

import (
	"bytes"
	"reflect"
	"sort"
	"testing"
	"testing/fstest"
	tpl "text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator/template"
)

func TestLoadTemplates(t *testing.T) {
	const LoremLipsum = "Lorem lipsum"

	tests := []struct {
		name        string
		args        fstest.MapFS
		baseDirName string
		want        []template.File
		wantErr     bool
	}{
		{
			name:        "single file template",
			args:        fstest.MapFS{"template/file": {Data: []byte(LoremLipsum)}},
			baseDirName: "template",
			want:        []template.File{{Name: "file", Path: ".", Template: GetTemplate("file", LoremLipsum)}},
		},
		{
			name: "multiple file template",
			args: fstest.MapFS{
				"template/file":           {Data: []byte(LoremLipsum)},
				"template/dir/file2":      {Data: []byte(LoremLipsum)},
				"template/dir/file1":      {Data: []byte(LoremLipsum)},
				"template/dir1/dir2/file": {Data: []byte(LoremLipsum)},
				"template/.hidden/file":   {Data: []byte(LoremLipsum)},
			},
			baseDirName: "template",
			want: []template.File{
				{Name: "file", Path: ".", Template: GetTemplate("file", LoremLipsum)},
				{Name: "file1", Path: "dir", Template: GetTemplate("file1", LoremLipsum)},
				{Name: "file2", Path: "dir", Template: GetTemplate("file2", LoremLipsum)},
				{Name: "file", Path: "dir1/dir2", Template: GetTemplate("file", LoremLipsum)},
				{Name: "file", Path: ".hidden", Template: GetTemplate("file", LoremLipsum)},
			},
			wantErr: false,
		},
		{
			name:        "missing template dir",
			args:        fstest.MapFS{"file": {Data: []byte(LoremLipsum)}},
			baseDirName: "template",
			want:        []template.File{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := template.LoadTemplates(tt.args, tt.baseDirName)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) != len(got) {
				t.Errorf("Different length for LoadTemplates() got = %v, want %v", got, tt.want)
				return
			}

			sort.Sort(template.Templates(tt.want))
			sort.Sort(got)

			for i := range tt.want {
				if !reflect.DeepEqual(got[i].Path, tt.want[i].Path) {
					t.Errorf("Different Path LoadTemplates() got = %v, want %v", got[i], tt.want[i])
					return
				}

				if !reflect.DeepEqual(got[i].Name, tt.want[i].Name) {
					t.Errorf("Different Name LoadTemplates() got = %v, want %v", got[i], tt.want[i])
					return
				}

				var b1, b2 bytes.Buffer
				gotTpl := got[i].Template.Execute(&b1, nil)
				wantTpl := tt.want[i].Template.Execute(&b2, nil)

				if b1.String() != b2.String() {
					t.Errorf("Different tamplate LoadTemplates() got = %v, want %v", gotTpl, wantTpl)
					return
				}
			}
		})
	}
}

func GetTemplate(name, text string) *tpl.Template {
	tpl, _ := tpl.New(name).
		Funcs(sprig.TxtFuncMap()).Parse(text)
	return tpl
}
