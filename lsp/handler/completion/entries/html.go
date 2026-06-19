package entries

import (
	// glsp "github.com/tliron/glsp"
	// server "github.com/tliron/glsp/server"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var HtmlTagCompletions = func(i []protocol.CompletionItem) []protocol.CompletionItem {
	mode := protocol.InsertTextModeAsIs
	kind := protocol.CompletionItemKindInterface
	items := i
	htmltags := map[string]string{
		"a":        "<a href=\"\"></a>",
		"p":        "<p></p>",
		"h1":       "<h1></h1>",
		"h2":       "<h2></h2>",
		"h3":       "<h3></h3>",
		"h4":       "<h4></h4>",
		"h5":       "<h5></h5>",
		"h6":       "<h6></h6>",
		"ul":       "<ul></ul>",
		"ol":       "<ol></ol>",
		"li":       "<li></li>",
		"nav":      "<nav></nav>",
		"t":        "<t></t>",
		"dl":       "<dl></dl>",
		"dt":       "<dt></dt>",
		"dd":       "<dd></dd>",
		"table":    "<table></table>",
		"tr":       "<tr></tr>",
		"td":       "<td></td>",
		"th":       "<th></th>",
		"thead":    "<thead></thead>",
		"tbody":    "<tbody></tbody>",
		"tfoot":    "<tfoot></tfoot>",
		"i":        "<i></i>",
		"hr":       "<hr/>",
		"br":       "<br/>",
		"em":       "<em></em>",
		"div":      "<div></div>",
		"img":      "<img src=\"\" alt=\"\"/>",
		"canvas":   "<canvas></canvas>",
		"script":   "<script></script>",
		"style":    "<style></style>",
		"meta":     "<meta name=\"\" content=\"\"/>",
		"title":    "<title></title>",
		"link":     "<link rel=\"\" href=\"\"/>",
		"footer":   "<footer></footer>",
		"header":   "<header></header>",
		"main":     "<main></main>",
		"section":  "<section></section>",
		"article":  "<article></article>",
		"aside":    "<aside></aside>",
		"button":   "<button></button>",
		"form":     "<form></form>",
		"input":    "<input type=\"\" name=\"\"/>",
		"label":    "<label></label>",
		"select":   "<select></select>",
		"option":   "<option></option>",
		"textarea": "<textarea></textarea>",
		"span":     "<span></span>",
		"strong":   "<strong></strong>",
		"iframe":   "<iframe src=\"\"></iframe>",
		"pre":      "<pre></pre>",
		"code":     "<code></code>",
	}
	t := true
	for tag, ins := range htmltags {
		desc := ins
		fmt := protocol.InsertTextFormatPlainText
		items = append(items, protocol.CompletionItem{
			Label:            tag,
			Documentation:    "# " + tag + "\n\n" + ins,
			InsertText:       &ins,
			Preselect:        &t,
			InsertTextFormat: &fmt,
			CommitCharacters: []string{"<"},
			InsertTextMode:   &mode,
			Detail:           &desc,
			Kind:             &kind,
		})
	}
	return items
}
