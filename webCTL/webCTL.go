package webCTL

import (
	"ChatWire/cfg"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var tmpl *template.Template

func Init() {
	tmpl = template.Must(template.ParseFiles("webCTL/template.html"))
	cfg.WebInterface.LocalSettings = cfg.Local
	cfg.WebInterface.GlobalSettings = cfg.Global
	http.HandleFunc("/", serveTemplate)
	http.ListenAndServe(":8080", nil)
}

func generateForm(v interface{}) string {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	var formFields []string
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		// Escape both field name and value
		escapedFieldName := html.EscapeString(fieldName)

		// Skip fields with form:"-" tag
		if tag := fieldType.Tag.Get("form"); tag == "-" {
			continue
		}

		// Skip fields with json:"-" tag
		if tag := fieldType.Tag.Get("json"); tag == "-" {
			continue
		}

		// Check if field is read-only
		if tag := fieldType.Tag.Get("form"); tag == "RO" {
			escapedValue := html.EscapeString(fmt.Sprintf("%v", field.Interface()))
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><span>`+escapedValue+`</span><br>`)
			continue
		}

		// Check if field is hidden (mask the value with ***)
		if tag := fieldType.Tag.Get("form"); tag == "hidden" {
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><span>**********</span><br>`)
			continue
		}

		// Handle different field types for input generation
		switch field.Kind() {
		case reflect.String:
			escapedValue := html.EscapeString(field.String())
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="text" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"><br>`)
		case reflect.Int, reflect.Int32, reflect.Int64:
			escapedValue := strconv.Itoa(int(field.Int()))
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="number" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"><br>`)
		case reflect.Float32, reflect.Float64:
			escapedValue := strconv.FormatFloat(field.Float(), 'f', 2, 64)
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="number" step="0.01" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"><br>`)
		case reflect.Bool:
			checked := ""
			if field.Bool() {
				checked = "checked"
			}
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="checkbox" id="`+escapedFieldName+`" name="`+escapedFieldName+`" `+checked+`><br>`)
		case reflect.Struct:
			formFields = append(formFields, `<h2>`+escapedFieldName+`</h2>`+generateForm(field.Interface()))
		}
	}
	return strings.Join(formFields, "\n")
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Handle form submission here...
	}

	// Generate separate forms for local and global settings
	localForm := generateForm(cfg.WebInterface.LocalSettings)
	globalForm := generateForm(cfg.WebInterface.GlobalSettings)

	// Pass the forms as data to the template
	tmpl.Execute(w, map[string]interface{}{
		"LocalForm":  template.HTML(localForm),
		"GlobalForm": template.HTML(globalForm),
	})
}

func updateStructFromForm(v interface{}, form map[string][]string) {
	val := reflect.ValueOf(v).Elem() // Get the struct value
	typ := reflect.TypeOf(v).Elem()  // Get the struct type

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		// Check if the field is present in the form
		if formValue, ok := form[fieldName]; ok {
			switch field.Kind() {
			case reflect.String:
				field.SetString(formValue[0])
			case reflect.Int, reflect.Int32, reflect.Int64:
				if intValue, err := strconv.Atoi(formValue[0]); err == nil {
					field.SetInt(int64(intValue))
				}
			case reflect.Float32, reflect.Float64:
				if floatValue, err := strconv.ParseFloat(formValue[0], 64); err == nil {
					field.SetFloat(floatValue)
				}
			case reflect.Bool:
				field.SetBool(len(formValue) > 0) // Checkbox is present if it's checked
			}
		}
	}
}
