package models

import (
	"errors"
	"testing"

	"ecommerce/utils"
)

// TestSubtotalItem comprueba el calculo del importe de una linea del pedido.
func TestSubtotalItem(t *testing.T) {
	item, err := NuevoItem("P001", "Monitor", 189.99, 3)
	if err != nil {
		t.Fatalf("no deberia haber error al crear la linea: %v", err)
	}
	esperado := 569.97
	if item.Subtotal() != esperado {
		t.Errorf("subtotal esperado %.2f, se obtuvo %.2f", esperado, item.Subtotal())
	}
}

// TestNuevoItemCantidadInvalida comprueba que no se acepten lineas con cantidad
// cero o negativa.
func TestNuevoItemCantidadInvalida(t *testing.T) {
	if _, err := NuevoItem("P001", "Monitor", 189.99, 0); err == nil {
		t.Error("deberia rechazarse una cantidad de cero")
	}
	if _, err := NuevoItem("P001", "Monitor", 189.99, -5); err == nil {
		t.Error("deberia rechazarse una cantidad negativa")
	}
}

// TestAgregarItemAcumula comprueba que agregar dos veces el mismo producto suma
// la cantidad en una sola linea en lugar de crear una linea repetida.
func TestAgregarItemAcumula(t *testing.T) {
	pedido := NuevoPedido("O001", "C001")

	item1, _ := NuevoItem("P001", "Teclado", 45.50, 2)
	item2, _ := NuevoItem("P001", "Teclado", 45.50, 3)

	if err := pedido.AgregarItem(item1); err != nil {
		t.Fatalf("no deberia fallar al agregar la primera linea: %v", err)
	}
	if err := pedido.AgregarItem(item2); err != nil {
		t.Fatalf("no deberia fallar al agregar la segunda linea: %v", err)
	}

	if len(pedido.Items()) != 1 {
		t.Errorf("se esperaba 1 linea acumulada, se obtuvieron %d", len(pedido.Items()))
	}
	if pedido.Items()[0].Cantidad() != 5 {
		t.Errorf("cantidad esperada 5, se obtuvo %d", pedido.Items()[0].Cantidad())
	}
}

// TestConfirmarPedidoVacio comprueba que no se pueda confirmar un pedido sin
// productos.
func TestConfirmarPedidoVacio(t *testing.T) {
	pedido := NuevoPedido("O001", "C001")

	err := pedido.Confirmar()
	if err == nil {
		t.Fatal("se esperaba error al confirmar un pedido vacio")
	}
	if !errors.Is(err, utils.ErrPedidoVacio) {
		t.Errorf("se esperaba ErrPedidoVacio, se obtuvo: %v", err)
	}
}

// TestPedidoNoSeModificaTrasConfirmar comprueba la maquina de estados: una vez
// confirmado, el pedido ya no admite cambios.
func TestPedidoNoSeModificaTrasConfirmar(t *testing.T) {
	pedido := NuevoPedido("O001", "C001")
	item, _ := NuevoItem("P001", "Teclado", 45.50, 2)
	pedido.AgregarItem(item)

	if err := pedido.Confirmar(); err != nil {
		t.Fatalf("confirmar un pedido con lineas no deberia fallar: %v", err)
	}
	if pedido.Estado() != EstadoConfirmado {
		t.Errorf("estado esperado CONFIRMADO, se obtuvo '%s'", pedido.Estado())
	}

	// Ya confirmado, agregar otra linea debe rechazarse.
	otro, _ := NuevoItem("P002", "Monitor", 189.99, 1)
	err := pedido.AgregarItem(otro)
	if err == nil {
		t.Fatal("no deberia poder agregarse una linea a un pedido confirmado")
	}
	if !errors.Is(err, utils.ErrPedidoCerrado) {
		t.Errorf("se esperaba ErrPedidoCerrado, se obtuvo: %v", err)
	}

	// Confirmar dos veces tampoco debe permitirse.
	if err := pedido.Confirmar(); err == nil {
		t.Error("no deberia poder confirmarse dos veces el mismo pedido")
	}
}

// TestSerializacionJSON comprueba que un producto se convierta a JSON y pueda
// reconstruirse conservando todos sus datos.
//
// Es una prueba de ida y vuelta: se serializa el objeto, se deserializa el
// resultado, y se verifica que el objeto reconstruido sea igual al original.
// Si MarshalJSON y UnmarshalJSON no fueran coherentes entre si, esta prueba
// fallaria.
func TestSerializacionJSON(t *testing.T) {
	original, _ := NuevoProducto("P001", "Teclado mecanico", 45.50, 20, 5)

	datos, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("no deberia fallar la serializacion: %v", err)
	}

	var reconstruido Producto
	if err := reconstruido.UnmarshalJSON(datos); err != nil {
		t.Fatalf("no deberia fallar la deserializacion: %v", err)
	}

	if reconstruido.ID() != original.ID() {
		t.Errorf("ID: se esperaba '%s', se obtuvo '%s'", original.ID(), reconstruido.ID())
	}
	if reconstruido.Nombre() != original.Nombre() {
		t.Errorf("nombre: se esperaba '%s', se obtuvo '%s'", original.Nombre(), reconstruido.Nombre())
	}
	if reconstruido.Precio() != original.Precio() {
		t.Errorf("precio: se esperaba %.2f, se obtuvo %.2f", original.Precio(), reconstruido.Precio())
	}
	if reconstruido.Stock() != original.Stock() {
		t.Errorf("stock: se esperaba %d, se obtuvo %d", original.Stock(), reconstruido.Stock())
	}
}

// TestUnmarshalRechazaDatosInvalidos comprueba que un archivo JSON editado a
// mano con valores invalidos no genere objetos corruptos.
func TestUnmarshalRechazaDatosInvalidos(t *testing.T) {
	jsonCorrupto := []byte(`{"id":"P001","nombre":"Teclado","precio":-999,"stock":10,"stock_minimo":2}`)

	var producto Producto
	err := producto.UnmarshalJSON(jsonCorrupto)
	if err == nil {
		t.Fatal("deberia rechazarse un JSON con precio negativo")
	}
	if !errors.Is(err, utils.ErrPrecioInvalido) {
		t.Errorf("se esperaba ErrPrecioInvalido, se obtuvo: %v", err)
	}
}

// TestPolimorfismoEntity comprueba que los tres tipos cumplen la interfaz
// Entity y que cada uno ejecuta su propia version de Describir.
func TestPolimorfismoEntity(t *testing.T) {
	producto, _ := NuevoProducto("P001", "Teclado", 45.50, 10, 3)
	cliente, _ := NuevoCliente("C001", "Maria Lopez", "maria@correo.com", "0991234567")
	pedido := NuevoPedido("O001", "C001")

	// Los tres se guardan en una sola lista de tipo Entity. Si alguno no
	// cumpliera la interfaz, el programa no compilaria.
	entidades := []Entity{producto, cliente, pedido}

	if len(entidades) != 3 {
		t.Fatalf("se esperaban 3 entidades, se obtuvieron %d", len(entidades))
	}

	// Cada elemento debe devolver una descripcion propia y distinta.
	descripciones := make(map[string]bool)
	for _, entidad := range entidades {
		descripcion := entidad.Describir()
		if descripcion == "" {
			t.Errorf("la entidad %s devolvio una descripcion vacia", entidad.ID())
		}
		if descripciones[descripcion] {
			t.Errorf("dos entidades distintas devolvieron la misma descripcion")
		}
		descripciones[descripcion] = true
	}
}
