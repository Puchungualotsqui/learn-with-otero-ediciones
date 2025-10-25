package helper

var MonthsES = [...]string{
	"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
	"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
}

func MonthNameES(month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return MonthsES[month]
}
