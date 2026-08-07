package models

import (
	"fmt"
	"strings"

	"ecommerce/utils"
)

// Producto representa un articulo del catalogo.
// El inventario esta dentro del producto: el campo stock guarda las unidades
// disponibles y stockMinimo el limite que dispara la alerta de stock bajo.
type Producto struct {
	id          string // campo privado: solo models puede tocarlo
	nombre      string
	precio      float64
	stock       int
	stockMinimo int
}

// NuevoProducto es el constructor. Valida los datos ANTES de crear el objeto,
// asi nunca existe un producto con precio negativo o sin nombre.
func NuevoProducto(id, nombre string, precio float64, stock, stockMinimo int) (*Producto, error) {
	nombre = strings.TrimSpace(nombre) // quita espacios al inicio y al final
	if len(nombre) < 3 {
		return nil, utils.ErrNombreInvalido
	}
	if precio <= 0 {
		return nil, utils.ErrPrecioInvalido
	}
	if stock < 0 || stockMinimo < 0 {
		return nil, utils.ErrStockNegativo
	}
	// &Producto{...} crea el producto y devuelve su direccion de memoria.
	// El segundo valor (nil) significa "no hubo error".
	return &Producto{
		id:          id,
		nombre:      nombre,
		precio:      precio,
		stock:       stock,
		stockMinimo: stockMinimo,
	}, nil
}

// ----- GETTERS: la unica forma de leer los campos privados desde fuera -----

// ID devuelve el codigo del producto.
func (p *Producto) ID() string { return p.id }

// Nombre devuelve el nombre del producto.
func (p *Producto) Nombre() string { return p.nombre }

// Precio devuelve el precio unitario.
func (p *Producto) Precio() float64 { return p.precio }

// Stock devuelve las unidades disponibles.
func (p *Producto) Stock() int { return p.stock }

// StockMinimo devuelve el limite de alerta.
func (p *Producto) StockMinimo() int { return p.stockMinimo }

// ----- SETTERS: modifican el objeto, pero validan primero -----

// CambiarNombre modifica el nombre si tiene al menos 3 caracteres.
func (p *Producto) CambiarNombre(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if len(nombre) < 3 {
		return utils.ErrNombreInvalido
	}
	p.nombre = nombre
	return nil
}

// CambiarPrecio modifica el precio si es mayor a cero.
func (p *Producto) CambiarPrecio(precio float64) error {
	if precio <= 0 {
		return utils.ErrPrecioInvalido
	}
	p.precio = precio
	return nil
}

// CambiarStock fija el stock en un valor exacto (ajuste de inventario).
func (p *Producto) CambiarStock(stock int) error {
	if stock < 0 {
		return utils.ErrStockNegativo
	}
	p.stock = stock
	return nil
}

// CambiarStockMinimo modifica el limite que dispara la alerta.
func (p *Producto) CambiarStockMinimo(minimo int) error {
	if minimo < 0 {
		return utils.ErrStockNegativo
	}
	p.stockMinimo = minimo
	return nil
}

// ----- COMPORTAMIENTO -----

// AumentarStock suma unidades al inventario (reposicion).
func (p *Producto) AumentarStock(cantidad int) error {
	if cantidad <= 0 {
		return utils.ErrCantidadInvalida
	}
	p.stock += cantidad
	return nil
}

// DescontarStock resta unidades al inventario (venta confirmada).
// Devuelve error si se pide mas de lo que hay.
func (p *Producto) DescontarStock(cantidad int) error {
	if cantidad <= 0 {
		return utils.ErrCantidadInvalida
	}
	if cantidad > p.stock {
		// %w mete el error original dentro del mensaje nuevo, para que
		// despues se pueda reconocer con errors.Is en el menu.
		return fmt.Errorf("%w: %s (hay %d, se pidieron %d)",
			utils.ErrStockInsuficiente, p.nombre, p.stock, cantidad)
	}
	p.stock -= cantidad
	return nil
}

// HayStock indica si alcanza el inventario para la cantidad pedida.
func (p *Producto) HayStock(cantidad int) bool {
	return cantidad > 0 && p.stock >= cantidad
}

// StockBajo indica si el producto llego al limite de alerta.
func (p *Producto) StockBajo() bool { return p.stock <= p.stockMinimo }

// Describir implementa la interfaz Entity para Producto.
func (p *Producto) Describir() string {
	estado := "disponible"
	if p.stock == 0 {
		estado = "AGOTADO"
	} else if p.StockBajo() {
		estado = "STOCK BAJO"
	}
	return fmt.Sprintf("[PRODUCTO] %-5s %-22s $%8.2f  stock:%4d  (%s)",
		p.id, p.nombre, p.precio, p.stock, estado)
}
