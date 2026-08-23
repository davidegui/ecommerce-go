// Sistema de Gestion de E-Commerce - Proyecto Final.
//
// Aplicacion desarrollada en Go que puede usarse de dos formas:
//  1. Como aplicacion de consola, con menus interactivos.
//  2. Como servidor web, exponiendo sus funcionalidades como servicios web
//     que responden en formato JSON.
//
// Las dos formas usan exactamente la misma logica de negocio, que vive en el
// paquete services. Los datos se guardan en archivos JSON dentro de la carpeta
// data, asi que se conservan entre ejecuciones.
//
// Este archivo solo se ocupa de la interaccion con el usuario por consola.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ecommerce/api"
	"ecommerce/models"
	"ecommerce/services"
	"ecommerce/utils"
)

// puertoServidor es el puerto donde escucha el servidor web.
const puertoServidor = 8080

// lector lee lo que el usuario escribe en el teclado.
var lector = bufio.NewReader(os.Stdin)

// main carga los datos guardados y muestra el menu principal.
func main() {
	// Al iniciar se leen los archivos JSON de la carpeta data. Si no existen,
	// se crean vacios automaticamente.
	if err := services.CargarDatos(); err != nil {
		fmt.Println("Error al cargar los datos guardados:", err)
		return
	}
	fmt.Println("Datos cargados desde la carpeta data/")

	for {
		fmt.Println("\n========================================")
		fmt.Println("   SISTEMA DE GESTION DE E-COMMERCE")
		fmt.Println("========================================")
		fmt.Println(" 1. Gestion de productos")
		fmt.Println(" 2. Gestion de clientes")
		fmt.Println(" 3. Gestion de pedidos")
		fmt.Println(" 4. Gestion de inventario")
		fmt.Println(" 5. Reportes concurrentes")
		fmt.Println(" 6. Ver todo el sistema (polimorfismo)")
		fmt.Println(" 7. Iniciar servidor web (servicios web)")
		fmt.Println(" 0. Salir")

		switch leerTexto("Opcion: ") {
		case "1":
			menuProductos()
		case "2":
			menuClientes()
		case "3":
			menuPedidos()
		case "4":
			menuInventario()
		case "5":
			menuReportes()
		case "6":
			verTodoElSistema()
		case "7":
			iniciarServidor()
		case "0":
			fmt.Println("Programa finalizado.")
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// iniciarServidor levanta el servidor web.
// Una vez iniciado, el programa se queda atendiendo peticiones y solo termina
// con Ctrl+C, por eso esta opcion no regresa al menu.
func iniciarServidor() {
	if err := api.Iniciar(puertoServidor); err != nil {
		fmt.Println("Error al iniciar el servidor:", err)
	}
}

// ==========================================================================
// FUNCIONES PARA LEER DATOS DEL TECLADO
// ==========================================================================

// leerTexto muestra un mensaje y devuelve lo que escribio el usuario.
func leerTexto(mensaje string) string {
	fmt.Print(mensaje)
	linea, _ := lector.ReadString('\n')
	// TrimSpace quita el salto de linea que queda al final.
	return strings.TrimSpace(linea)
}

// leerEntero lee un numero entero del teclado.
func leerEntero(mensaje string) (int, error) {
	texto := leerTexto(mensaje)
	numero, err := strconv.Atoi(texto) // Atoi convierte texto a entero
	if err != nil {
		return 0, fmt.Errorf("'%s' no es un numero entero", texto)
	}
	return numero, nil
}

// leerDecimal lee un numero con decimales del teclado.
func leerDecimal(mensaje string) (float64, error) {
	// Se cambia la coma por punto porque aqui se escribe 45,50 y Go espera 45.50
	texto := strings.Replace(leerTexto(mensaje), ",", ".", 1)
	numero, err := strconv.ParseFloat(texto, 64)
	if err != nil {
		return 0, fmt.Errorf("'%s' no es un numero valido", texto)
	}
	return numero, nil
}

// mostrarError imprime el error y, si lo reconoce, agrega una sugerencia.
//
// errors.Is revisa si el error recibido contiene el error que se busca, aunque
// venga envuelto con informacion extra por el verbo %w.
func mostrarError(err error) {
	fmt.Println(">> ERROR:", err)
	if errors.Is(err, utils.ErrStockInsuficiente) {
		fmt.Println("   Sugerencia: reponga inventario o pida menos unidades.")
	}
	if errors.Is(err, utils.ErrProductoNoEncontrado) {
		fmt.Println("   Sugerencia: revise el listado de productos.")
	}
	if errors.Is(err, utils.ErrClienteNoEncontrado) {
		fmt.Println("   Sugerencia: registre primero al cliente.")
	}
	if errors.Is(err, utils.ErrArchivo) {
		fmt.Println("   Sugerencia: verifique los permisos de la carpeta data/.")
	}
}

// ==========================================================================
// MODULO DE PRODUCTOS
// ==========================================================================

func menuProductos() {
	for {
		fmt.Println("\n--- PRODUCTOS ---")
		fmt.Println(" 1. Registrar producto")
		fmt.Println(" 2. Modificar producto")
		fmt.Println(" 3. Eliminar producto")
		fmt.Println(" 4. Consultar productos disponibles")
		fmt.Println(" 5. Buscar producto por nombre")
		fmt.Println(" 0. Volver")

		switch leerTexto("Opcion: ") {
		case "1":
			registrarProducto()
		case "2":
			modificarProducto()
		case "3":
			eliminarProducto()
		case "4":
			mostrarProductos(services.ConsultarDisponibles())
		case "5":
			mostrarProductos(services.BuscarPorNombre(leerTexto("Texto a buscar: ")))
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// registrarProducto pide los datos y registra un producto nuevo.
func registrarProducto() {
	nombre := leerTexto("Nombre: ")
	precio, err := leerDecimal("Precio: ")
	if err != nil {
		mostrarError(err)
		return
	}
	stock, err := leerEntero("Stock inicial: ")
	if err != nil {
		mostrarError(err)
		return
	}
	minimo, err := leerEntero("Stock minimo (para alertas): ")
	if err != nil {
		mostrarError(err)
		return
	}
	producto, err := services.RegistrarProducto(nombre, precio, stock, minimo)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Registrado:", producto.Describir())
}

// modificarProducto cambia los datos de un producto existente.
// Los campos que el usuario deja en blanco no se modifican.
func modificarProducto() {
	id := leerTexto("Codigo del producto: ")
	fmt.Println("(deje en blanco lo que no quiera cambiar)")

	nombre := leerTexto("Nuevo nombre: ")

	precio := 0.0
	if texto := leerTexto("Nuevo precio: "); texto != "" {
		valor, err := strconv.ParseFloat(strings.Replace(texto, ",", ".", 1), 64)
		if err != nil {
			mostrarError(fmt.Errorf("'%s' no es un numero valido", texto))
			return
		}
		precio = valor
	}

	minimo := -1 // -1 significa "no cambiar el stock minimo"
	if texto := leerTexto("Nuevo stock minimo: "); texto != "" {
		valor, err := strconv.Atoi(texto)
		if err != nil {
			mostrarError(fmt.Errorf("'%s' no es un numero entero", texto))
			return
		}
		minimo = valor
	}

	producto, err := services.ModificarProducto(id, nombre, precio, minimo)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Modificado:", producto.Describir())
}

// eliminarProducto quita un producto del catalogo.
func eliminarProducto() {
	id := leerTexto("Codigo del producto a eliminar: ")
	if err := services.EliminarProducto(id); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Producto eliminado.")
}

// mostrarProductos imprime una lista de productos.
func mostrarProductos(lista []*models.Producto) {
	if len(lista) == 0 {
		fmt.Println("No hay productos para mostrar.")
		return
	}
	fmt.Println()
	for _, producto := range lista {
		fmt.Println(producto.Describir())
	}
}

// ==========================================================================
// MODULO DE CLIENTES
// ==========================================================================

func menuClientes() {
	for {
		fmt.Println("\n--- CLIENTES ---")
		fmt.Println(" 1. Registrar cliente")
		fmt.Println(" 2. Actualizar informacion")
		fmt.Println(" 3. Consultar clientes registrados")
		fmt.Println(" 4. Eliminar cliente")
		fmt.Println(" 0. Volver")

		switch leerTexto("Opcion: ") {
		case "1":
			registrarCliente()
		case "2":
			actualizarCliente()
		case "3":
			mostrarClientes()
		case "4":
			eliminarCliente()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// registrarCliente pide los datos y registra un cliente nuevo.
func registrarCliente() {
	nombre := leerTexto("Nombre: ")
	email := leerTexto("Correo: ")
	telefono := leerTexto("Telefono: ")

	cliente, err := services.RegistrarCliente(nombre, email, telefono)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Registrado:", cliente.Describir())
}

// actualizarCliente cambia los datos de un cliente ya registrado.
func actualizarCliente() {
	id := leerTexto("Codigo del cliente: ")
	fmt.Println("(deje en blanco lo que no quiera cambiar)")
	nombre := leerTexto("Nuevo nombre: ")
	email := leerTexto("Nuevo correo: ")
	telefono := leerTexto("Nuevo telefono: ")

	cliente, err := services.ActualizarCliente(id, nombre, email, telefono)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Actualizado:", cliente.Describir())
}

// mostrarClientes imprime todos los clientes registrados.
func mostrarClientes() {
	lista := services.ListarClientes()
	if len(lista) == 0 {
		fmt.Println("No hay clientes registrados.")
		return
	}
	fmt.Println()
	for _, cliente := range lista {
		fmt.Println(cliente.Describir())
	}
}

// eliminarCliente quita un cliente del sistema.
func eliminarCliente() {
	id := leerTexto("Codigo del cliente a eliminar: ")
	if err := services.EliminarCliente(id); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Cliente eliminado.")
}

// ==========================================================================
// MODULO DE PEDIDOS
// ==========================================================================

func menuPedidos() {
	for {
		fmt.Println("\n--- PEDIDOS ---")
		fmt.Println(" 1. Crear pedido")
		fmt.Println(" 2. Agregar producto al pedido")
		fmt.Println(" 3. Calcular total y aplicar descuento")
		fmt.Println(" 4. Confirmar pedido")
		fmt.Println(" 5. Listar pedidos")
		fmt.Println(" 0. Volver")

		switch leerTexto("Opcion: ") {
		case "1":
			crearPedido()
		case "2":
			agregarProductoAPedido()
		case "3":
			verTotalPedido()
		case "4":
			confirmarPedido()
		case "5":
			listarPedidos()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// crearPedido crea un pedido vacio para un cliente existente.
func crearPedido() {
	pedido, err := services.CrearPedido(leerTexto("Codigo del cliente: "))
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Pedido creado:", pedido.Describir())
}

// agregarProductoAPedido agrega una linea al detalle del pedido.
func agregarProductoAPedido() {
	pedidoID := leerTexto("Codigo del pedido: ")
	productoID := leerTexto("Codigo del producto: ")
	cantidad, err := leerEntero("Cantidad: ")
	if err != nil {
		mostrarError(err)
		return
	}
	if _, err := services.AgregarProductoAPedido(pedidoID, productoID, cantidad); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Producto agregado al pedido.")
}

// verTotalPedido calcula el total del pedido y muestra el detalle.
func verTotalPedido() {
	pedido, explicacion, err := services.AplicarDescuento(leerTexto("Codigo del pedido: "))
	if err != nil {
		mostrarError(err)
		return
	}
	mostrarDetalle(pedido, explicacion)
}

// confirmarPedido cierra el pedido, descuenta inventario y muestra alertas.
func confirmarPedido() {
	pedido, explicacion, alertas, err := services.ConfirmarPedido(leerTexto("Codigo del pedido: "))
	if err != nil {
		mostrarError(err)
		return
	}
	mostrarDetalle(pedido, explicacion)
	fmt.Println("PEDIDO CONFIRMADO. Inventario descontado.")
	if len(alertas) > 0 {
		fmt.Println("\n*** ALERTAS DE STOCK BAJO ***")
		for _, producto := range alertas {
			fmt.Printf("  - %s: quedan %d unidades (minimo %d)\n",
				producto.Nombre(), producto.Stock(), producto.StockMinimo())
		}
	}
}

// listarPedidos muestra todos los pedidos registrados.
func listarPedidos() {
	lista := services.ListarPedidos()
	if len(lista) == 0 {
		fmt.Println("No hay pedidos registrados.")
		return
	}
	fmt.Println()
	for _, pedido := range lista {
		fmt.Println(pedido.Describir())
	}
}

// mostrarDetalle imprime las lineas del pedido y el resumen de totales.
func mostrarDetalle(pedido *models.Pedido, explicacion string) {
	fmt.Printf("\nPedido %s - cliente %s - estado %s\n",
		pedido.ID(), pedido.ClienteID(), pedido.Estado())
	fmt.Println("--------------------------------------------------")
	for _, item := range pedido.Items() {
		fmt.Printf("  %-22s %3d x $%8.2f = $%9.2f\n",
			item.Nombre(), item.Cantidad(), item.Precio(), item.Subtotal())
	}
	fmt.Println("--------------------------------------------------")
	if explicacion != "" {
		fmt.Println("  " + explicacion)
	}
	fmt.Printf("  Subtotal : $%9.2f\n", pedido.Subtotal())
	fmt.Printf("  Descuento: %.0f%%\n", pedido.Descuento())
	fmt.Printf("  TOTAL    : $%9.2f\n", pedido.Total())
}

// ==========================================================================
// MODULO DE INVENTARIO
// ==========================================================================

func menuInventario() {
	for {
		fmt.Println("\n--- INVENTARIO ---")
		fmt.Println(" 1. Consultar stock de un producto")
		fmt.Println(" 2. Actualizar existencias (valor exacto)")
		fmt.Println(" 3. Reponer existencias (sumar)")
		fmt.Println(" 4. Ver alertas de stock bajo")
		fmt.Println(" 0. Volver")

		switch leerTexto("Opcion: ") {
		case "1":
			consultarStock()
		case "2":
			actualizarStock()
		case "3":
			reponerStock()
		case "4":
			verAlertas()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// consultarStock muestra el estado de inventario de un producto.
func consultarStock() {
	producto, err := services.BuscarProducto(leerTexto("Codigo del producto: "))
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Printf("%s (%s): %d unidades, minimo %d\n",
		producto.ID(), producto.Nombre(), producto.Stock(), producto.StockMinimo())
	if producto.StockBajo() {
		fmt.Println("  -> ALERTA DE STOCK BAJO")
	}
}

// actualizarStock fija las existencias en un valor exacto.
func actualizarStock() {
	id := leerTexto("Codigo del producto: ")
	cantidad, err := leerEntero("Nuevo stock: ")
	if err != nil {
		mostrarError(err)
		return
	}
	producto, err := services.ActualizarStock(id, cantidad)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Stock actualizado:", producto.Describir())
}

// reponerStock suma unidades al inventario de un producto.
func reponerStock() {
	id := leerTexto("Codigo del producto: ")
	cantidad, err := leerEntero("Unidades a ingresar: ")
	if err != nil {
		mostrarError(err)
		return
	}
	producto, err := services.ReponerStock(id, cantidad)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Inventario repuesto:", producto.Describir())
}

// verAlertas muestra los productos que llegaron al limite minimo.
func verAlertas() {
	alertas := services.AlertasStockBajo()
	if len(alertas) == 0 {
		fmt.Println("No hay alertas de stock bajo.")
		return
	}
	fmt.Println("\n*** PRODUCTOS CON STOCK BAJO ***")
	for _, producto := range alertas {
		fmt.Printf("  - %-22s stock:%3d  minimo:%3d\n",
			producto.Nombre(), producto.Stock(), producto.StockMinimo())
	}
}

// ==========================================================================
// MODULO DE REPORTES CONCURRENTES
// ==========================================================================

func menuReportes() {
	for {
		fmt.Println("\n--- REPORTES ---")
		fmt.Println(" 1. Generar los tres reportes en paralelo")
		fmt.Println(" 2. Monitorear alertas de stock")
		fmt.Println(" 0. Volver")

		switch leerTexto("Opcion: ") {
		case "1":
			generarReportes()
		case "2":
			monitorearStock()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// generarReportes lanza los tres reportes al mismo tiempo y muestra los
// resultados junto con el tiempo que tardo cada uno.
func generarReportes() {
	inicio := time.Now()
	reportes := services.GenerarReportes()
	total := time.Since(inicio)

	sumaIndividual := time.Duration(0)
	for _, reporte := range reportes {
		fmt.Println("\n============================================")
		fmt.Println(" " + reporte.Titulo)
		fmt.Println("============================================")
		if reporte.Error != "" {
			fmt.Println("  ERROR:", reporte.Error)
			continue
		}
		for _, linea := range reporte.Lineas {
			fmt.Println("  " + linea)
		}
		fmt.Printf("  (calculado en %v)\n", reporte.Demora)
		sumaIndividual += reporte.Demora
	}

	// Esta comparacion es la que demuestra que los reportes se calcularon en
	// paralelo: el tiempo total es cercano al del reporte mas lento, no a la
	// suma de los tres.
	fmt.Println("\n--------------------------------------------")
	fmt.Printf("  Suma de los tres calculos : %v\n", sumaIndividual)
	fmt.Printf("  Tiempo real transcurrido  : %v\n", total)
	fmt.Println("  Los tres reportes se calcularon en paralelo.")
}

// monitorearStock recibe las alertas por un canal a medida que se detectan.
//
// La funcion MonitorearStock devuelve el canal de inmediato y sigue revisando
// el inventario por detras. El bucle range lee del canal hasta que este se
// cierra, mostrando cada alerta en cuanto llega.
func monitorearStock() {
	fmt.Println("\nMonitoreando inventario...")

	encontradas := 0
	for alerta := range services.MonitorearStock() {
		fmt.Println("  " + alerta)
		encontradas++
	}

	if encontradas == 0 {
		fmt.Println("  Sin alertas: todo el inventario esta en niveles normales.")
		return
	}
	fmt.Printf("  Total de alertas: %d\n", encontradas)
}

// ==========================================================================
// DEMOSTRACION DE POLIMORFISMO
// ==========================================================================

// verTodoElSistema imprime productos, clientes y pedidos con un solo bucle.
//
// La funcion TodasLasEntidades devuelve una lista de tipo models.Entity que
// contiene los tres tipos mezclados. Cada elemento ejecuta su propia version
// del metodo Describir, y Go decide cual segun el tipo real de cada uno: eso
// es polimorfismo.
func verTodoElSistema() {
	lista := services.TodasLasEntidades()
	if len(lista) == 0 {
		fmt.Println("Todavia no hay datos cargados.")
		return
	}
	fmt.Println("\n--- TODOS LOS REGISTROS DEL SISTEMA ---")
	for _, entidad := range lista {
		fmt.Println(entidad.Describir())
	}
}
