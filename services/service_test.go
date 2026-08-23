package services

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"ecommerce/models"
	"ecommerce/storage"
	"ecommerce/utils"
)

// TestMain prepara el entorno antes de ejecutar las pruebas del paquete.
//
// Las pruebas escriben archivos, asi que se ejecutan en un directorio temporal
// para no tocar los datos reales de la carpeta data del proyecto.
func TestMain(m *testing.M) {
	temporal, err := os.MkdirTemp("", "pruebas-ecommerce")
	if err != nil {
		fmt.Println("no se pudo crear el directorio temporal:", err)
		os.Exit(1)
	}
	if err := os.Chdir(temporal); err != nil {
		fmt.Println("no se pudo cambiar de directorio:", err)
		os.Exit(1)
	}

	codigo := m.Run() // ejecuta todas las pruebas del paquete

	os.RemoveAll(temporal)
	os.Exit(codigo)
}

// limpiarDatos deja el sistema vacio antes de cada prueba, para que una prueba
// no dependa de lo que hizo la anterior.
func limpiarDatos() {
	candado.Lock()
	productos = []*models.Producto{}
	clientes = []*models.Cliente{}
	pedidos = []*models.Pedido{}
	contadorProductos = 0
	contadorClientes = 0
	contadorPedidos = 0
	candado.Unlock()

	storage.GuardarProductos(nil)
	storage.GuardarClientes(nil)
	storage.GuardarPedidos(nil)
}

// ==========================================================================
// PRUEBAS DEL CALCULO DE TOTALES Y DESCUENTOS
// ==========================================================================

// TestDescuentoEscalonado comprueba que se aplique el porcentaje correcto en
// cada tramo de monto.
//
// Los casos incluyen los limites exactos de cada tramo, que es donde suelen
// esconderse los errores: $99.99 no debe tener descuento pero $100.00 si.
func TestDescuentoEscalonado(t *testing.T) {
	casos := []struct {
		descripcion       string
		precio            float64
		cantidad          int
		subtotalEsperado  float64
		descuentoEsperado float64
		totalEsperado     float64
	}{
		{"por debajo del primer tramo", 30.00, 3, 90.00, 0, 90.00},
		{"justo por debajo de 100", 99.99, 1, 99.99, 0, 99.99},
		{"limite exacto de 100", 100.00, 1, 100.00, 5, 95.00},
		{"dentro del tramo del 5%", 50.00, 4, 200.00, 5, 190.00},
		{"limite exacto de 300", 300.00, 1, 300.00, 10, 270.00},
		{"dentro del tramo del 10%", 100.00, 5, 500.00, 10, 450.00},
		{"limite exacto de 700", 700.00, 1, 700.00, 15, 595.00},
		{"dentro del tramo del 15%", 400.00, 3, 1200.00, 15, 1020.00},
	}

	for _, caso := range casos {
		t.Run(caso.descripcion, func(t *testing.T) {
			limpiarDatos()

			// Se registra un producto con stock suficiente y un cliente.
			if _, err := RegistrarProducto("Producto de prueba", caso.precio, 100, 5); err != nil {
				t.Fatalf("no se pudo registrar el producto: %v", err)
			}
			if _, err := RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567"); err != nil {
				t.Fatalf("no se pudo registrar el cliente: %v", err)
			}
			pedido, err := CrearPedido("C001")
			if err != nil {
				t.Fatalf("no se pudo crear el pedido: %v", err)
			}
			// Los servicios devuelven una copia del pedido, no el objeto que
			// esta guardado en la lista. Por eso hay que usar el valor que
			// devuelve cada operacion y no la variable anterior.
			pedido, err = AgregarProductoAPedido(pedido.ID(), "P001", caso.cantidad)
			if err != nil {
				t.Fatalf("no se pudo agregar el producto: %v", err)
			}

			if pedido.Subtotal() != caso.subtotalEsperado {
				t.Errorf("subtotal: se esperaba %.2f, se obtuvo %.2f",
					caso.subtotalEsperado, pedido.Subtotal())
			}
			if pedido.Descuento() != caso.descuentoEsperado {
				t.Errorf("descuento: se esperaba %.0f%%, se obtuvo %.0f%%",
					caso.descuentoEsperado, pedido.Descuento())
			}
			if pedido.Total() != caso.totalEsperado {
				t.Errorf("total: se esperaba %.2f, se obtuvo %.2f",
					caso.totalEsperado, pedido.Total())
			}
		})
	}
}

// TestCalcularTotalPedidoVacio comprueba que un pedido sin lineas no se pueda
// calcular.
func TestCalcularTotalPedidoVacio(t *testing.T) {
	limpiarDatos()
	RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567")
	pedido, _ := CrearPedido("C001")

	_, err := CalcularTotal(pedido)
	if err == nil {
		t.Fatal("se esperaba error al calcular un pedido vacio")
	}
	if !errors.Is(err, utils.ErrPedidoVacio) {
		t.Errorf("se esperaba ErrPedidoVacio, se obtuvo: %v", err)
	}
}

// ==========================================================================
// PRUEBAS DE VALIDACION DE STOCK
// ==========================================================================

// TestNoSePuedeAgregarMasDelStock comprueba que el pedido rechace cantidades
// mayores al inventario disponible.
func TestNoSePuedeAgregarMasDelStock(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Mouse", 25.00, 3, 5)
	RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567")
	pedido, _ := CrearPedido("C001")

	_, err := AgregarProductoAPedido(pedido.ID(), "P001", 99)
	if err == nil {
		t.Fatal("se esperaba error al pedir 99 unidades habiendo solo 3")
	}
	if !errors.Is(err, utils.ErrStockInsuficiente) {
		t.Errorf("se esperaba ErrStockInsuficiente, se obtuvo: %v", err)
	}
	// Se vuelve a consultar el pedido para comprobar que no quedo modificado.
	guardado, _ := BuscarPedido(pedido.ID())
	if len(guardado.Items()) != 0 {
		t.Error("el pedido no deberia tener lineas cuando la operacion falla")
	}
}

// TestConfirmarDescuentaInventario comprueba que confirmar un pedido descuente
// las existencias de los productos vendidos.
func TestConfirmarDescuentaInventario(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Monitor", 189.99, 10, 2)
	RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567")
	pedido, _ := CrearPedido("C001")
	AgregarProductoAPedido(pedido.ID(), "P001", 4)

	confirmado, _, _, err := ConfirmarPedido(pedido.ID())
	if err != nil {
		t.Fatalf("confirmar no deberia fallar: %v", err)
	}

	producto, _ := BuscarProducto("P001")
	if producto.Stock() != 6 {
		t.Errorf("stock esperado 6 tras vender 4 de 10, se obtuvo %d", producto.Stock())
	}
	if confirmado.Estado() != models.EstadoConfirmado {
		t.Errorf("estado esperado CONFIRMADO, se obtuvo '%s'", confirmado.Estado())
	}
}

// TestConfirmarGeneraAlertas comprueba que la venta dispare la alerta de stock
// bajo cuando el producto queda por debajo del minimo.
func TestConfirmarGeneraAlertas(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Monitor", 189.99, 8, 5)
	RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567")
	pedido, _ := CrearPedido("C001")
	AgregarProductoAPedido(pedido.ID(), "P001", 4)

	_, _, alertas, err := ConfirmarPedido(pedido.ID())
	if err != nil {
		t.Fatalf("confirmar no deberia fallar: %v", err)
	}
	// Quedan 4 unidades y el minimo es 5, asi que debe alertar.
	if len(alertas) != 1 {
		t.Errorf("se esperaba 1 alerta de stock bajo, se obtuvieron %d", len(alertas))
	}
}

// ==========================================================================
// PRUEBAS DE CONCURRENCIA
// ==========================================================================

// TestRegistroConcurrente comprueba que el candado protege las listas cuando
// varias goroutines registran productos al mismo tiempo.
//
// Se lanzan 50 goroutines que registran un producto cada una. Si el acceso no
// estuviera protegido, algunos registros se perderian al escribir dos
// goroutines sobre la lista a la vez.
//
// Esta prueba tambien se puede ejecutar con el detector de condiciones de
// carrera que trae Go:
//
//	go test -race ./services/
func TestRegistroConcurrente(t *testing.T) {
	limpiarDatos()

	const cantidad = 50
	var grupo sync.WaitGroup

	for i := 0; i < cantidad; i++ {
		grupo.Add(1)
		go func(numero int) {
			defer grupo.Done()
			RegistrarProducto(fmt.Sprintf("Producto %d", numero), 10.00, 5, 2)
		}(i)
	}
	grupo.Wait() // espera a que las 50 terminen

	registrados := len(ListarProductos())
	if registrados != cantidad {
		t.Errorf("se esperaban %d productos registrados, se obtuvieron %d: "+
			"se perdieron registros por acceso concurrente", cantidad, registrados)
	}
}

// TestReportesConcurrentes comprueba que los tres reportes se generen
// correctamente usando goroutines y canales.
func TestReportesConcurrentes(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Teclado", 45.50, 20, 5)
	RegistrarProducto("Monitor", 189.99, 8, 2)
	RegistrarCliente("Cliente Prueba", "prueba@correo.com", "0991234567")
	pedido, _ := CrearPedido("C001")
	AgregarProductoAPedido(pedido.ID(), "P001", 2)
	ConfirmarPedido(pedido.ID())

	reportes := GenerarReportes()

	if len(reportes) != 3 {
		t.Fatalf("se esperaban 3 reportes, se obtuvieron %d", len(reportes))
	}
	for _, reporte := range reportes {
		if reporte.Error != "" {
			t.Errorf("el reporte '%s' devolvio error: %s", reporte.Titulo, reporte.Error)
		}
		if len(reporte.Lineas) == 0 {
			t.Errorf("el reporte '%s' no devolvio ninguna linea", reporte.Titulo)
		}
	}
}

// TestMonitorStockPorCanal comprueba que el monitor envie las alertas por el
// canal y que el canal se cierre al terminar.
func TestMonitorStockPorCanal(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Con stock normal", 10.00, 50, 5)
	RegistrarProducto("Con stock bajo", 10.00, 2, 5)
	RegistrarProducto("Agotado", 10.00, 0, 5)

	alertas := []string{}
	// range sobre el canal termina cuando el canal se cierra. Si el monitor
	// no lo cerrara, esta prueba se quedaria bloqueada para siempre.
	for alerta := range MonitorearStock() {
		alertas = append(alertas, alerta)
	}

	// Se esperan dos alertas: el de stock bajo y el agotado.
	if len(alertas) != 2 {
		t.Errorf("se esperaban 2 alertas, se obtuvieron %d: %v", len(alertas), alertas)
	}
}

// ==========================================================================
// PRUEBAS DE BUSQUEDA
// ==========================================================================

// TestBuscarPorNombre comprueba la busqueda parcial sin distinguir mayusculas.
func TestBuscarPorNombre(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Monitor 24 pulgadas", 189.99, 8, 2)
	RegistrarProducto("Teclado mecanico", 45.50, 20, 5)
	RegistrarProducto("Mouse inalambrico", 25.00, 15, 3)

	casos := []struct {
		texto     string
		esperados int
	}{
		{"monitor", 1},
		{"MONITOR", 1},
		{"mo", 2}, // Monitor y Mouse
		{"inexistente", 0},
	}

	for _, caso := range casos {
		t.Run("buscar "+caso.texto, func(t *testing.T) {
			encontrados := BuscarPorNombre(caso.texto)
			if len(encontrados) != caso.esperados {
				t.Errorf("buscando '%s' se esperaban %d resultados, se obtuvieron %d",
					caso.texto, caso.esperados, len(encontrados))
			}
		})
	}
}

// TestBuscarProductoInexistente comprueba el error cuando el codigo no existe.
func TestBuscarProductoInexistente(t *testing.T) {
	limpiarDatos()

	_, err := BuscarProducto("P999")
	if err == nil {
		t.Fatal("se esperaba error al buscar un producto que no existe")
	}
	if !errors.Is(err, utils.ErrProductoNoEncontrado) {
		t.Errorf("se esperaba ErrProductoNoEncontrado, se obtuvo: %v", err)
	}
}

// ==========================================================================
// PRUEBAS DE PERSISTENCIA
// ==========================================================================

// TestPersistenciaJSON comprueba que los datos guardados se puedan recuperar.
//
// Se registran productos, se vacia la memoria simulando un reinicio, y se
// vuelven a cargar desde el archivo JSON.
func TestPersistenciaJSON(t *testing.T) {
	limpiarDatos()
	RegistrarProducto("Teclado mecanico", 45.50, 20, 5)
	RegistrarCliente("Maria Lopez", "maria@correo.com", "0991234567")

	// Se vacian las listas en memoria, como si el programa se hubiera cerrado.
	candado.Lock()
	productos = []*models.Producto{}
	clientes = []*models.Cliente{}
	candado.Unlock()

	if err := CargarDatos(); err != nil {
		t.Fatalf("no se pudieron cargar los datos: %v", err)
	}

	if len(ListarProductos()) != 1 {
		t.Errorf("se esperaba 1 producto tras recargar, se obtuvieron %d",
			len(ListarProductos()))
	}
	if len(ListarClientes()) != 1 {
		t.Errorf("se esperaba 1 cliente tras recargar, se obtuvieron %d",
			len(ListarClientes()))
	}

	producto, err := BuscarProducto("P001")
	if err != nil {
		t.Fatalf("el producto guardado deberia existir tras recargar: %v", err)
	}
	if producto.Nombre() != "Teclado mecanico" || producto.Precio() != 45.50 {
		t.Errorf("los datos no se conservaron correctamente: %s a %.2f",
			producto.Nombre(), producto.Precio())
	}
}
