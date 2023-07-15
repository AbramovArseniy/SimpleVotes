package templates

import "html/template"

var IndexTemplate = template.Must(template.ParseFiles("./internal/templates/html/index.html"))
var UnauthorizedTemplate = template.Must(template.ParseFiles("./internal/templates/html/unauthorized.html"))
var AuthTemplate = template.Must(template.ParseFiles("./internal/templates/html/authorization.html"))
var AddQuestionTemplate = template.Must(template.ParseFiles("./internal/templates/html/add-question.html"))
var ProfileTemplate = template.Must(template.ParseFiles("./internal/templates/html/profile.html"))
