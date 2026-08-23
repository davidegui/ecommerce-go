package services

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"ecommerce/models"
)

// ==========================================================================
// REPORTES CONCURRENTES
// ==========================================================================
//
// Este archivo genera los reportes del sistema usando goroutines y canales.
//
// Una goroutine es una funcion que se ejecuta al mismo tiempo que el resto del
// programa, sin esperar a que termine. Se lanza escribiendo la palabra go
// delante de la llamada.
//
// Un canal es una tuberia por la que las goroutines se envian datos entre si.
// Se crea con make(chan tipo) y funciona en los dos sentidos: una goroutine
// escribe con canal <- dato, y otra lee con dato := <-canal.
//
// Los tres reportes de este archivo son independientes entre si: ninguno
// necesita el resultado de los otros. Por eso pueden calcularse en paralelo en
// lugar de uno detras de otro.

// Reporte contiene el resultado de un reporte generado.
//
// El campo Error permite que una goroutine informe que su calculo fallo sin
// detener a las demas. Como cada goroutine se ejecuta por separado, no puede
// simplemente devolver un error: tiene que enviarlo por el canal junto con el
// resto del resultado.
type Reporte struct {
	Titulo  string        `json:"titulo"`
	Lineas  []string      `json:"lineas"`
	Error   string        `json:"error,omitempty"`
	Demora  time.Duration `json:"-"`
	DemoraT string        `json:"tiempo_calculo"`
}

// GenerarReportes calcula los tres reportes del sistema en paralelo.
//
// Funcionamiento paso a paso:
//
//  1. Se crea un canal con capacidad para tres reportes.
//  2. Se lanzan tres goroutines, una por reporte. Las tres empiezan a trabajar
//     al mismo tiempo.
//  3. Un WaitGroup lleva la cuenta de cuantas goroutines faltan por terminar.
//  4. Una cuarta goroutine espera a que las tres terminen y cierra el canal.
//  5. El bucle final lee del canal hasta que se cierra, recogiendo los
//     resultados en el orden en que van llegando.
//
// El tiempo total es el del reporte mas lento, no la suma de los tres.
func GenerarReportes() []Reporte {
	// Canal con espacio para tres resultados. Al tener capacidad, las
	// goroutines pueden depositar su resultado y terminar sin quedarse
	// esperando a que alguien lo recoja.
	canal := make(chan Reporte, 3)

	// El WaitGroup es un contador de goroutines pendientes.
	var grupo sync.WaitGroup

	// Add(1) suma una goroutine al contador antes de lanzarla.
	grupo.Add(1)
	go func() {
		// defer grupo.Done() resta uno al contador cuando la goroutine
		// termina, sin importar como termine.
		defer grupo.Done()
		canal <- reporteVentas()
	}()

	grupo.Add(1)
	go func() {
		defer grupo.Done()
		canal <- reporteMasVendidos()
	}()

	grupo.Add(1)
	go func() {
		defer grupo.Done()
		canal <- reporteInventario()
	}()

	// Esta goroutine espera a que el contador llegue a cero y recien entonces
	// cierra el canal. Se hace en una goroutine aparte porque grupo.Wait()
	// bloquea, y si se llamara aqui directamente el programa se detendria
	// antes de empezar a leer los resultados.
	go func() {
		grupo.Wait()
		close(canal)
	}()

	// range sobre un canal lee valores hasta que el canal se cierra.
	resultados := []Reporte{}
	for reporte := range canal {
		resultados = append(resultados, reporte)
	}

	// Los resultados llegan en orden impredecible, porque depende de cual
	// goroutine termine primero. Se ordenan por titulo para que el reporte
	// salga siempre igual.
	sort.Slice(resultados, func(i, j int) bool {
		return resultados[i].Titulo < resultados[j].Titulo
	})
	return resultados
}

// reporteVentas resume los pedidos confirmados y el monto total facturado.
func reporteVentas() Reporte {
	inicio := time.Now()

	candado.Lock()
	copiaPedidos := make([]*models.Pedido, len(pedidos))
	copy(copiaPedidos, pedidos)
	candado.Unlock()

	confirmados := 0
	pendientes := 0
	facturado := 0.0
	descuentos := 0.0

	for _, pedido := range copiaPedidos {
		if pedido.Estado() == models.EstadoConfirmado {
			confirmados++
			facturado += pedido.Total()
			descuentos += pedido.Subtotal() - pedido.Total()
		} else {
			pendientes++
		}
	}

	lineas := []string{
		fmt.Sprintf("Pedidos confirmados : %d", confirmados),
		fmt.Sprintf("Pedidos pendientes  : %d", pendientes),
		fmt.Sprintf("Total facturado     : $%.2f", facturado),
		fmt.Sprintf("Total en descuentos : $%.2f", descuentos),
	}
	if confirmados > 0 {
		lineas = append(lineas,
			fmt.Sprintf("Ticket promedio     : $%.2f", facturado/float64(confirmados)))
	}

	return construirReporte("1. Ventas del periodo", lineas, inicio)
}

// reporteMasVendidos ordena los productos por unidades vendidas.
//
// Aqui se usa un map para acumular las unidades de cada producto: la clave es
// el codigo y el valor las unidades acumuladas. El map es la estructura
// adecuada porque permite sumar directamente sobre la clave sin tener que
// recorrer una lista buscando el producto en cada linea.
func reporteMasVendidos() Reporte {
	inicio := time.Now()

	candado.Lock()
	copiaPedidos := make([]*models.Pedido, len(pedidos))
	copy(copiaPedidos, pedidos)
	candado.Unlock()

	// map[codigo]unidades y map[codigo]monto
	unidades := make(map[string]int)
	montos := make(map[string]float64)
	nombres := make(map[string]string)

	for _, pedido := range copiaPedidos {
		// Solo cuentan los pedidos confirmados: un pedido pendiente todavia
		// no es una venta.
		if pedido.Estado() != models.EstadoConfirmado {
			continue
		}
		for _, item := range pedido.Items() {
			unidades[item.ProductoID()] += item.Cantidad()
			montos[item.ProductoID()] += item.Subtotal()
			nombres[item.ProductoID()] = item.Nombre()
		}
	}

	if len(unidades) == 0 {
		return construirReporte("2. Productos mas vendidos",
			[]string{"Todavia no hay ventas confirmadas."}, inicio)
	}

	// Para ordenar hay que pasar el map a un slice, porque los maps de Go no
	// tienen orden: recorrerlos con range devuelve las claves en orden
	// aleatorio a proposito.
	type fila struct {
		codigo   string
		nombre   string
		unidades int
		monto    float64
	}
	filas := []fila{}
	for codigo, cantidad := range unidades {
		filas = append(filas, fila{codigo, nombres[codigo], cantidad, montos[codigo]})
	}
	// sort.Slice ordena usando la funcion de comparacion que se le pasa.
	sort.Slice(filas, func(i, j int) bool {
		return filas[i].unidades > filas[j].unidades
	})

	lineas := []string{}
	for posicion, f := range filas {
		lineas = append(lineas, fmt.Sprintf("%d. %-22s %3d unidades   $%9.2f",
			posicion+1, f.nombre, f.unidades, f.monto))
	}
	return construirReporte("2. Productos mas vendidos", lineas, inicio)
}

// reporteInventario resume el estado del stock del catalogo.
func reporteInventario() Reporte {
	inicio := time.Now()

	candado.Lock()
	copiaProductos := make([]*models.Producto, len(productos))
	copy(copiaProductos, productos)
	candado.Unlock()

	if len(copiaProductos) == 0 {
		return construirReporte("3. Estado del inventario",
			[]string{"No hay productos registrados."}, inicio)
	}

	agotados := 0
	enAlerta := 0
	valorTotal := 0.0
	for _, producto := range copiaProductos {
		valorTotal += producto.Precio() * float64(producto.Stock())
		if producto.Stock() == 0 {
			agotados++
		} else if producto.StockBajo() {
			enAlerta++
		}
	}

	lineas := []string{
		fmt.Sprintf("Productos en catalogo : %d", len(copiaProductos)),
		fmt.Sprintf("Productos agotados    : %d", agotados),
		fmt.Sprintf("Productos en alerta   : %d", enAlerta),
		fmt.Sprintf("Valor del inventario  : $%.2f", valorTotal),
	}
	return construirReporte("3. Estado del inventario", lineas, inicio)
}

// construirReporte arma el resultado con el tiempo que tardo el calculo.
func construirReporte(titulo string, lineas []string, inicio time.Time) Reporte {
	demora := time.Since(inicio)
	return Reporte{
		Titulo:  titulo,
		Lineas:  lineas,
		Demora:  demora,
		DemoraT: demora.String(),
	}
}

// ==========================================================================
// MONITOR DE ALERTAS DE STOCK
// ==========================================================================

// MonitorearStock revisa el inventario en una goroutine independiente y envia
// las alertas encontradas por un canal, a medida que las va detectando.
//
// A diferencia de GenerarReportes, esta funcion NO espera a que el trabajo
// termine: devuelve el canal de inmediato y sigue trabajando por detras. Quien
// la llama puede ir mostrando las alertas conforme llegan, sin quedarse
// bloqueado hasta el final.
//
// El canal se devuelve como <-chan string, o sea un canal de solo lectura. Asi
// el compilador impide que quien lo recibe escriba en el por error.
func MonitorearStock() <-chan string {
	canal := make(chan string)

	go func() {
		// Cerrar el canal al terminar es obligatorio: es la senal de que no
		// van a llegar mas datos. Sin close, quien lee con range se quedaria
		// esperando para siempre.
		defer close(canal)

		candado.Lock()
		copiaProductos := make([]*models.Producto, len(productos))
		copy(copiaProductos, productos)
		candado.Unlock()

		for _, producto := range copiaProductos {
			if producto.Stock() == 0 {
				canal <- fmt.Sprintf("AGOTADO    | %-22s sin existencias", producto.Nombre())
			} else if producto.StockBajo() {
				canal <- fmt.Sprintf("STOCK BAJO | %-22s quedan %d (minimo %d)",
					producto.Nombre(), producto.Stock(), producto.StockMinimo())
			}
		}
	}()

	return canal
}
