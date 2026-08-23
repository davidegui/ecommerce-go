package models

import (
	"errors"
	"testing"

	"ecommerce/utils"
)

// Las pruebas unitarias se escriben en archivos terminados en _test.go y se
// ejecutan con el comando: go test ./...
//
// Cada prueba es una funcion que empieza con Test y recibe *testing.T. Cuando
// algo no sale como se esperaba se llama a t.Errorf, que marca la prueba como
// fallida y muestra el mensaje.

// TestNuevoProductoValido comprueba que un producto con datos correctos se crea
// sin errores y guarda los valores recibidos.
func TestNuevoProductoValido(t *testing.T) {
	producto, err := NuevoProducto("P001", "Teclado mecanico", 45.50, 20, 5)
	if err != nil {
		t.Fatalf("no deberia haber error al crear un producto valido, se obtuvo: %v", err)
	}
	if producto.Nombre() != "Teclado mecanico" {
		t.Errorf("nombre esperado 'Teclado mecanico', se obtuvo '%s'", producto.Nombre())
	}
	if producto.Precio() != 45.50 {
		t.Errorf("precio esperado 45.50, se obtuvo %.2f", producto.Precio())
	}
	if producto.Stock() != 20 {
		t.Errorf("stock esperado 20, se obtuvo %d", producto.Stock())
	}
}

// TestNuevoProductoInvalido comprueba que el constructor rechaza los datos que
// violan las reglas de negocio.
//
// Se usa una tabla de casos: cada elemento describe una entrada y el error que
// se espera recibir. Es la forma habitual de escribir pruebas en Go, porque
// agregar un caso nuevo es agregar una linea en lugar de una funcion entera.
func TestNuevoProductoInvalido(t *testing.T) {
	casos := []struct {
		descripcion   string
		nombre        string
		precio        float64
		stock         int
		stockMinimo   int
		errorEsperado error
	}{
		{"nombre demasiado corto", "ab", 10, 5, 1, utils.ErrNombreInvalido},
		{"nombre vacio", "   ", 10, 5, 1, utils.ErrNombreInvalido},
		{"precio en cero", "Teclado", 0, 5, 1, utils.ErrPrecioInvalido},
		{"precio negativo", "Teclado", -15, 5, 1, utils.ErrPrecioInvalido},
		{"stock negativo", "Teclado", 10, -1, 1, utils.ErrStockNegativo},
		{"stock minimo negativo", "Teclado", 10, 5, -3, utils.ErrStockNegativo},
	}

	for _, caso := range casos {
		// t.Run crea una subprueba con nombre propio, asi el reporte indica
		// exactamente cual de los casos fallo.
		t.Run(caso.descripcion, func(t *testing.T) {
			producto, err := NuevoProducto("P001", caso.nombre, caso.precio, caso.stock, caso.stockMinimo)
			if err == nil {
				t.Fatalf("se esperaba un error y el producto se creo igual")
			}
			if !errors.Is(err, caso.errorEsperado) {
				t.Errorf("error esperado '%v', se obtuvo '%v'", caso.errorEsperado, err)
			}
			if producto != nil {
				t.Errorf("cuando hay error el producto debe ser nil")
			}
		})
	}
}

// TestDescontarStock comprueba el descuento de inventario por una venta.
func TestDescontarStock(t *testing.T) {
	producto, _ := NuevoProducto("P001", "Monitor", 189.99, 10, 2)

	if err := producto.DescontarStock(3); err != nil {
		t.Fatalf("descontar 3 de 10 no deberia fallar: %v", err)
	}
	if producto.Stock() != 7 {
		t.Errorf("stock esperado 7, se obtuvo %d", producto.Stock())
	}
}

// TestDescontarStockInsuficiente comprueba que no se puede vender mas de lo que
// hay, y que el stock no se modifica cuando la operacion es rechazada.
func TestDescontarStockInsuficiente(t *testing.T) {
	producto, _ := NuevoProducto("P001", "Monitor", 189.99, 3, 2)

	err := producto.DescontarStock(99)
	if err == nil {
		t.Fatal("se esperaba error al pedir 99 unidades habiendo solo 3")
	}
	// errors.Is encuentra el error original aunque venga envuelto con el
	// nombre del producto y las cantidades.
	if !errors.Is(err, utils.ErrStockInsuficiente) {
		t.Errorf("se esperaba ErrStockInsuficiente, se obtuvo: %v", err)
	}
	if producto.Stock() != 3 {
		t.Errorf("el stock no debe cambiar cuando la operacion falla, quedo en %d", producto.Stock())
	}
}

// TestAumentarStock comprueba la reposicion de inventario.
func TestAumentarStock(t *testing.T) {
	producto, _ := NuevoProducto("P001", "Mouse", 25, 5, 3)

	if err := producto.AumentarStock(20); err != nil {
		t.Fatalf("reponer 20 unidades no deberia fallar: %v", err)
	}
	if producto.Stock() != 25 {
		t.Errorf("stock esperado 25, se obtuvo %d", producto.Stock())
	}
	// Reponer una cantidad negativa no tiene sentido y debe rechazarse.
	if err := producto.AumentarStock(-5); err == nil {
		t.Error("se esperaba error al reponer una cantidad negativa")
	}
}

// TestStockBajo comprueba la regla que dispara las alertas de inventario.
func TestStockBajo(t *testing.T) {
	casos := []struct {
		descripcion string
		stock       int
		minimo      int
		esperado    bool
	}{
		{"stock por encima del minimo", 10, 5, false},
		{"stock igual al minimo", 5, 5, true},
		{"stock por debajo del minimo", 2, 5, true},
		{"producto agotado", 0, 5, true},
	}

	for _, caso := range casos {
		t.Run(caso.descripcion, func(t *testing.T) {
			producto, _ := NuevoProducto("P001", "Producto", 10, caso.stock, caso.minimo)
			if producto.StockBajo() != caso.esperado {
				t.Errorf("con stock %d y minimo %d se esperaba %v",
					caso.stock, caso.minimo, caso.esperado)
			}
		})
	}
}

// TestSettersValidan comprueba que los setters rechazan valores invalidos y
// dejan el objeto sin modificar.
func TestSettersValidan(t *testing.T) {
	producto, _ := NuevoProducto("P001", "Teclado", 45.50, 10, 3)

	if err := producto.CambiarPrecio(-100); err == nil {
		t.Error("CambiarPrecio deberia rechazar un precio negativo")
	}
	if producto.Precio() != 45.50 {
		t.Errorf("el precio no debe cambiar si el setter rechaza el valor, quedo en %.2f",
			producto.Precio())
	}

	if err := producto.CambiarNombre("ab"); err == nil {
		t.Error("CambiarNombre deberia rechazar un nombre de menos de 3 caracteres")
	}
	if producto.Nombre() != "Teclado" {
		t.Errorf("el nombre no debe cambiar si el setter rechaza el valor, quedo en '%s'",
			producto.Nombre())
	}

	// Un valor valido si debe aplicarse.
	if err := producto.CambiarPrecio(59.99); err != nil {
		t.Errorf("CambiarPrecio deberia aceptar un precio valido: %v", err)
	}
	if producto.Precio() != 59.99 {
		t.Errorf("precio esperado 59.99, se obtuvo %.2f", producto.Precio())
	}
}
