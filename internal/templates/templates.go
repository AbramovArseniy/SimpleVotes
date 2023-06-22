package templates

import "html/template"

var IndexTemplate = template.Must(template.ParseFiles("../internal/templates/html/index.html"))
