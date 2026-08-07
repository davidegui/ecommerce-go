// Package utils contiene los errores del sistema en un solo lugar.
package utils

import "errors"

// Errores de validacion de datos.
var (
	ErrNombreInvalido   = errors.New("el nombre debe tener al menos 3 caracteres")
	ErrPrecioInvalido   = errors.New("el precio debe ser mayor a cero")
	ErrStockNegativo    = errors.New("el stock no puede ser negativo")
	ErrCantidadInvalida = errors.New("la cantidad debe ser mayor a cero")
	ErrEmailInvalido    = errors.New("el correo no tiene un formato valido")
)

// Errores de busqueda y de negocio.
var (
	ErrProductoNoEncontrado = errors.New("producto no encontrado")
	ErrClienteNoEncontrado  = errors.New("cliente no encontrado")
	ErrPedidoNoEncontrado   = errors.New("pedido no encontrado")
	ErrStockInsuficiente    = errors.New("stock insuficiente")
	ErrPedidoVacio          = errors.New("el pedido no tiene productos")
	ErrPedidoCerrado        = errors.New("el pedido ya fue confirmado y no se puede modificar")
)
