package webCTL

import (
	"ChatWire/cfg"
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

func CensorString(s string) string {
	// Convert the string to a slice of runes to handle Unicode characters correctly
	runes := []rune(s)
	asterisks := strings.Repeat("*", len(runes))
	return asterisks
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

		// Skip fields with json:"-" tag
		if tag := fieldType.Tag.Get("web"); tag != "" {
			escapedFieldName = html.EscapeString(tag)
		}

		// Check if field is read-only
		extraTag := ""
		if tag := fieldType.Tag.Get("form"); tag == "RO" {
			if tag := fieldType.Tag.Get("form"); tag == "RO" {
				extraTag = " disable; readonly"
			}
		}

		// Check if field is hidden (mask the value with ***)
		hide := false
		if tag := fieldType.Tag.Get("form"); tag == "hidden" {
			hide = true
		}

		// Handle different field types for input generation
		switch field.Kind() {
		case reflect.String:
			escapedValue := html.EscapeString(field.String())
			if hide {
				escapedValue = CensorString(escapedValue)
			}
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="text" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"`+extraTag+`><br>`)
		case reflect.Int, reflect.Int32, reflect.Int64:
			escapedValue := strconv.Itoa(int(field.Int()))
			if hide {
				escapedValue = CensorString(escapedValue)
			}
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="number" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"`+extraTag+`><br>`)
		case reflect.Float32, reflect.Float64:
			escapedValue := strconv.FormatFloat(field.Float(), 'f', 2, 64)
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="number" step="0.01" id="`+escapedFieldName+`" name="`+escapedFieldName+`" value="`+escapedValue+`"><br>`)
		case reflect.Bool:
			checked := ""
			if field.Bool() && !hide {
				checked = "checked"
			}
			formFields = append(formFields, `<label for="`+escapedFieldName+`">`+escapedFieldName+`:</label><input type="checkbox" id="`+escapedFieldName+`" name="`+escapedFieldName+`" `+checked+``+extraTag+`><br>`)
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
