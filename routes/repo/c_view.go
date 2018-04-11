package repo

import (
	"github.com/gogits/gogs/pkg/tool"
	"github.com/gogits/git-module"
	"github.com/gogits/gogs/pkg/template/highlight"
	"github.com/gogits/gogs/pkg/setting"
	"github.com/gogits/gogs/pkg/markup"
	"github.com/gogits/gogs/pkg/template"
	 log "gopkg.in/clog.v1"
	"io/ioutil"
	"path"
	"bytes"
	"strings"
	"fmt"
	"github.com/gogits/gogs/pkg/context"
	 gotemplate "html/template"
)

func C_renderFile(c *context.Context, entry *git.TreeEntry, treeLink, rawLink string) {
	c.Data["IsViewFile"] = true

	blob := entry.Blob()
	dataRc, err := blob.Data()
	if err != nil {
		c.Handle(500, "Data", err)
		return
	}

	c.Data["FileSize"] = blob.Size()
	c.Data["FileName"] = blob.Name()
	c.Data["HighlightClass"] = highlight.FileNameToHighlightClass(blob.Name())
	c.Data["RawFileLink"] = rawLink + "/" + c.Repo.TreePath

	buf := make([]byte, 1024)
	n, _ := dataRc.Read(buf)
	buf = buf[:n]

	isTextFile := tool.IsTextFile(buf)
	c.Data["IsTextFile"] = isTextFile

	// Assume file is not editable first.
	if !isTextFile {
		c.Data["EditFileTooltip"] = c.Tr("repo.editor.cannot_edit_non_text_files")
	}

	canEnableEditor := c.Repo.CanEnableEditor()
	switch {
	case isTextFile:
		if blob.Size() >= setting.UI.MaxDisplayFileSize {
			c.Data["IsFileTooLarge"] = true
			break
		}

		c.Data["ReadmeExist"] = markup.IsReadmeFile(blob.Name())

		d, _ := ioutil.ReadAll(dataRc)
		buf = append(buf, d...)

		switch markup.Detect(blob.Name()) {
		case markup.MARKDOWN:
			c.Data["IsMarkdown"] = true
			c.Data["FileContent"] = string(markup.Markdown(buf, path.Dir(treeLink), c.Repo.Repository.ComposeMetas()))
		case markup.ORG_MODE:
			c.Data["IsMarkdown"] = true
			c.Data["FileContent"] = string(markup.OrgMode(buf, path.Dir(treeLink), c.Repo.Repository.ComposeMetas()))
		case markup.IPYTHON_NOTEBOOK:
			c.Data["IsIPythonNotebook"] = true
		default:
			// Building code view blocks with line number on server side.
			var fileContent string
			if err, content := template.ToUTF8WithErr(buf); err != nil {
				if err != nil {
					log.Error(4, "ToUTF8WithErr: %s", err)
				}
				fileContent = string(buf)
			} else {
				fileContent = content
			}

			var output bytes.Buffer
			lines := strings.Split(fileContent, "\n")
			for index, line := range lines {
				output.WriteString(fmt.Sprintf(`<li class="L%d" rel="L%d">%s</li>`, index+1, index+1, gotemplate.HTMLEscapeString(strings.TrimRight(line, "\r"))) + "\n")
			}
			c.Data["FileContent"] = gotemplate.HTML(output.String())

			output.Reset()
			for i := 0; i < len(lines); i++ {
				output.WriteString(fmt.Sprintf(`<span id="L%d">%d</span>`, i+1, i+1))
			}
			c.Data["LineNums"] = gotemplate.HTML(output.String())
		}

		if canEnableEditor {
			c.Data["CanEditFile"] = true
			c.Data["EditFileTooltip"] = c.Tr("repo.editor.edit_this_file")
		} else if !c.Repo.IsViewBranch {
			c.Data["EditFileTooltip"] = c.Tr("repo.editor.must_be_on_a_branch")
		} else if !c.Repo.IsWriter() {
			c.Data["EditFileTooltip"] = c.Tr("repo.editor.fork_before_edit")
		}

	case tool.IsPDFFile(buf):
		c.Data["IsPDFFile"] = true
	case tool.IsVideoFile(buf):
		c.Data["IsVideoFile"] = true
	case tool.IsImageFile(buf):
		c.Data["IsImageFile"] = true
	}

	if canEnableEditor {
		c.Data["CanDeleteFile"] = true
		c.Data["DeleteFileTooltip"] = c.Tr("repo.editor.delete_this_file")
	} else if !c.Repo.IsViewBranch {
		c.Data["DeleteFileTooltip"] = c.Tr("repo.editor.must_be_on_a_branch")
	} else if !c.Repo.IsWriter() {
		c.Data["DeleteFileTooltip"] = c.Tr("repo.editor.must_have_write_access")
	}
}

