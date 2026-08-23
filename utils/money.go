package utils

import "strconv"

// RedondearDinero deja un valor monetario con dos decimales.
//
// Es necesario porque los numeros float64 se guardan en binario y muchos
// decimales no tienen representacion exacta: por ejemplo 0.1 mas 0.2 da
// 0.30000000000000004. Al sumar varias lineas de un pedido ese error se
// acumula y el total terminaria mostrandose como 594.8730000000001.
//
// El redondeo se hace convirtiendo el numero a texto con dos decimales y
// volviendolo a convertir a numero. Se usa strconv y no el paquete math porque
// strconv ya se usa en el resto del proyecto.
func RedondearDinero(valor float64) float64 {
	redondeado, err := strconv.ParseFloat(strconv.FormatFloat(valor, 'f', 2, 64), 64)
	if err != nil {
		return valor
	}
	return redondeado
}
