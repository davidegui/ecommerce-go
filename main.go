// Sistema de Gestion de E-Commerce - aplicacion de consola en Go.
//
// Este archivo contiene los menus de la aplicacion, las listas donde se
// guardan los datos y la logica de negocio del sistema.
//
// Los datos se guardan en memoria mientras el programa esta abierto: al
// cerrarlo se pierden. La persistencia en archivos JSON queda para la entrega
// final del proyecto.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ecommerce/models"
	"ecommerce/utils"
)

// Listas donde se guardan los datos del sistema.
// Son slices de punteros: guardan la direccion de cada objeto, para que al
// modificarlo se modifique el que esta en la lista y no una copia.
var (
	productos []*models.Producto
	clientes  []*models.Cliente
	pedidos   []*models.Pedido
)

// Contadores para generar los codigos P001, C001, O001.
var (
	contadorProductos int
	contadorClientes  int
	contadorPedidos   int
)

// lector lee lo que el usuario escribe en el teclado.
var lector = bufio.NewReader(os.Stdin)

// Reglas de descuento de la empresa.
// Estan como constantes para que la regla de negocio este en un solo lugar y
// no como numeros sueltos dentro del calculo.
const (
	montoParaCincoPorciento  = 100.0
	montoParaDiezPorciento   = 300.0
	montoParaQuincePorciento = 700.0
)

// main muestra el menu principal hasta que el usuario elige salir.
func main() {
	for {
		fmt.Println("\n========================================")
		fmt.Println("   SISTEMA DE GESTION DE E-COMMERCE")
		fmt.Println("========================================")
		fmt.Println(" 1. Gestion de productos")
		fmt.Println(" 2. Gestion de clientes")
		fmt.Println(" 3. Gestion de pedidos")
		fmt.Println(" 4. Gestion de inventario")
		fmt.Println(" 5. Ver todo el sistema (polimorfismo)")
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
			verTodoElSistema()
		case "0":
			fmt.Println("Programa finalizado.")
			return
		default:
			fmt.Println("Opcion no valida.")
		}
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
}

// ==========================================================================
// FUNCIONES DE BUSQUEDA
// ==========================================================================

// buscarProducto devuelve el producto que tenga ese codigo.
func buscarProducto(id string) (*models.Producto, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	// range recorre la lista; el indice se ignora con _ porque no se usa.
	for _, producto := range productos {
		if producto.ID() == id {
			return producto, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrProductoNoEncontrado, id)
}

// buscarCliente devuelve el cliente que tenga ese codigo.
func buscarCliente(id string) (*models.Cliente, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, cliente := range clientes {
		if cliente.ID() == id {
			return cliente, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrClienteNoEncontrado, id)
}

// buscarPedido devuelve el pedido que tenga ese codigo.
func buscarPedido(id string) (*models.Pedido, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, pedido := range pedidos {
		if pedido.ID() == id {
			return pedido, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrPedidoNoEncontrado, id)
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
			consultarDisponibles()
		case "5":
			buscarPorNombre()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// registrarProducto pide los datos, crea el producto y lo agrega a la lista.
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

	contadorProductos++
	codigo := fmt.Sprintf("P%03d", contadorProductos) // %03d rellena: P001

	// El constructor valida los datos. Si algo esta mal devuelve error y aqui
	// se devuelve el codigo al contador para no dejar huecos en la numeracion.
	producto, err := models.NuevoProducto(codigo, nombre, precio, stock, minimo)
	if err != nil {
		contadorProductos--
		mostrarError(err)
		return
	}

	// append agrega al final de la lista y devuelve el slice nuevo,
	// por eso hay que reasignarlo.
	productos = append(productos, producto)
	fmt.Println("Registrado:", producto.Describir())
}

// modificarProducto cambia nombre, precio o stock minimo de un producto.
// Los datos que el usuario deja en blanco no se modifican.
func modificarProducto() {
	id := leerTexto("Codigo del producto: ")
	producto, err := buscarProducto(id)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("(deje en blanco lo que no quiera cambiar)")

	if nombre := leerTexto("Nuevo nombre: "); nombre != "" {
		if err := producto.CambiarNombre(nombre); err != nil {
			mostrarError(err)
			return
		}
	}
	if texto := leerTexto("Nuevo precio: "); texto != "" {
		precio, err := strconv.ParseFloat(strings.Replace(texto, ",", ".", 1), 64)
		if err != nil {
			mostrarError(fmt.Errorf("'%s' no es un numero valido", texto))
			return
		}
		if err := producto.CambiarPrecio(precio); err != nil {
			mostrarError(err)
			return
		}
	}
	if texto := leerTexto("Nuevo stock minimo: "); texto != "" {
		minimo, err := strconv.Atoi(texto)
		if err != nil {
			mostrarError(fmt.Errorf("'%s' no es un numero entero", texto))
			return
		}
		if err := producto.CambiarStockMinimo(minimo); err != nil {
			mostrarError(err)
			return
		}
	}
	fmt.Println("Modificado:", producto.Describir())
}

// eliminarProducto quita un producto de la lista.
func eliminarProducto() {
	id := strings.ToUpper(strings.TrimSpace(leerTexto("Codigo a eliminar: ")))
	for i, producto := range productos {
		if producto.ID() == id {
			// Para borrar de un slice se pegan las dos partes que quedan: lo
			// que esta antes del elemento y lo que esta despues. Los tres
			// puntos (...) desarman el segundo slice en elementos sueltos.
			productos = append(productos[:i], productos[i+1:]...)
			fmt.Println("Producto eliminado.")
			return
		}
	}
	mostrarError(fmt.Errorf("%w: %s", utils.ErrProductoNoEncontrado, id))
}

// consultarDisponibles muestra solo los productos que tienen stock.
func consultarDisponibles() {
	encontrados := 0
	fmt.Println()
	for _, producto := range productos {
		if producto.Stock() > 0 {
			fmt.Println(producto.Describir())
			encontrados++
		}
	}
	if encontrados == 0 {
		fmt.Println("No hay productos disponibles.")
	}
}

// buscarPorNombre busca productos cuyo nombre contenga el texto escrito.
// La comparacion se hace en minusculas para que no distinga mayusculas.
func buscarPorNombre() {
	texto := strings.ToLower(strings.TrimSpace(leerTexto("Texto a buscar: ")))
	encontrados := 0
	fmt.Println()
	for _, producto := range productos {
		if strings.Contains(strings.ToLower(producto.Nombre()), texto) {
			fmt.Println(producto.Describir())
			encontrados++
		}
	}
	if encontrados == 0 {
		fmt.Println("Sin coincidencias.")
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
			consultarClientes()
		case "4":
			eliminarCliente()
		case "0":
			return
		default:
			fmt.Println("Opcion no valida.")
		}
	}
}

// registrarCliente pide los datos, crea el cliente y lo agrega a la lista.
func registrarCliente() {
	nombre := leerTexto("Nombre: ")
	email := leerTexto("Correo: ")
	telefono := leerTexto("Telefono: ")

	contadorClientes++
	codigo := fmt.Sprintf("C%03d", contadorClientes)

	cliente, err := models.NuevoCliente(codigo, nombre, email, telefono)
	if err != nil {
		contadorClientes--
		mostrarError(err)
		return
	}
	clientes = append(clientes, cliente)
	fmt.Println("Registrado:", cliente.Describir())
}

// actualizarCliente cambia los datos de un cliente ya registrado.
func actualizarCliente() {
	id := leerTexto("Codigo del cliente: ")
	cliente, err := buscarCliente(id)
	if err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("(deje en blanco lo que no quiera cambiar)")

	if nombre := leerTexto("Nuevo nombre: "); nombre != "" {
		if err := cliente.CambiarNombre(nombre); err != nil {
			mostrarError(err)
			return
		}
	}
	if email := leerTexto("Nuevo correo: "); email != "" {
		if err := cliente.CambiarEmail(email); err != nil {
			mostrarError(err)
			return
		}
	}
	if telefono := leerTexto("Nuevo telefono: "); telefono != "" {
		if err := cliente.CambiarTelefono(telefono); err != nil {
			mostrarError(err)
			return
		}
	}
	fmt.Println("Actualizado:", cliente.Describir())
}

// consultarClientes muestra todos los clientes registrados.
func consultarClientes() {
	if len(clientes) == 0 {
		fmt.Println("No hay clientes registrados.")
		return
	}
	fmt.Println()
	for _, cliente := range clientes {
		fmt.Println(cliente.Describir())
	}
}

// eliminarCliente quita un cliente de la lista.
func eliminarCliente() {
	id := strings.ToUpper(strings.TrimSpace(leerTexto("Codigo a eliminar: ")))
	for i, cliente := range clientes {
		if cliente.ID() == id {
			clientes = append(clientes[:i], clientes[i+1:]...)
			fmt.Println("Cliente eliminado.")
			return
		}
	}
	mostrarError(fmt.Errorf("%w: %s", utils.ErrClienteNoEncontrado, id))
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

// crearPedido crea un pedido vacio para un cliente que exista.
func crearPedido() {
	id := leerTexto("Codigo del cliente: ")
	cliente, err := buscarCliente(id)
	if err != nil {
		mostrarError(err)
		return
	}
	contadorPedidos++
	codigo := fmt.Sprintf("O%03d", contadorPedidos)

	pedido := models.NuevoPedido(codigo, cliente.ID())
	pedidos = append(pedidos, pedido)
	fmt.Println("Pedido creado:", pedido.Describir())
}

// agregarProductoAPedido agrega una linea al detalle del pedido.
// Revisa que haya stock, pero NO descuenta inventario todavia: eso ocurre
// recien al confirmar el pedido.
func agregarProductoAPedido() {
	pedido, err := buscarPedido(leerTexto("Codigo del pedido: "))
	if err != nil {
		mostrarError(err)
		return
	}
	producto, err := buscarProducto(leerTexto("Codigo del producto: "))
	if err != nil {
		mostrarError(err)
		return
	}
	cantidad, err := leerEntero("Cantidad: ")
	if err != nil {
		mostrarError(err)
		return
	}
	if !producto.HayStock(cantidad) {
		mostrarError(fmt.Errorf("%w: %s (hay %d, se pidieron %d)",
			utils.ErrStockInsuficiente, producto.Nombre(), producto.Stock(), cantidad))
		return
	}

	item, err := models.NuevoItem(producto.ID(), producto.Nombre(), producto.Precio(), cantidad)
	if err != nil {
		mostrarError(err)
		return
	}
	if err := pedido.AgregarItem(item); err != nil {
		mostrarError(err)
		return
	}
	// Se recalcula el total para que el pedido quede siempre coherente.
	if _, err := calcularTotal(pedido); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Producto agregado al pedido.")
}

// calcularTotal es la funcion principal del sistema.
//
// Hace tres cosas: revisa que haya stock de cada producto del pedido, suma el
// subtotal, y aplica un descuento escalonado segun el monto. Devuelve un texto
// explicando el descuento y guarda los totales dentro del pedido.
//
// Si algun producto no tiene stock suficiente devuelve error y no modifica
// nada: es preferible rechazar el pedido completo a dejarlo mal calculado.
func calcularTotal(pedido *models.Pedido) (string, error) {
	// --- PASO 1: el pedido no puede estar vacio ---
	items := pedido.Items()
	if len(items) == 0 {
		return "", utils.ErrPedidoVacio
	}

	// --- PASO 2: sumar el subtotal y revisar el stock de cada linea ---
	subtotal := 0.0
	for _, item := range items {
		// Se busca el producto otra vez porque el stock pudo haber cambiado
		// desde que la linea se agrego al pedido.
		producto, err := buscarProducto(item.ProductoID())
		if err != nil {
			return "", err
		}
		if !producto.HayStock(item.Cantidad()) {
			return "", fmt.Errorf("%w: %s (hay %d, se pidieron %d)",
				utils.ErrStockInsuficiente, producto.Nombre(),
				producto.Stock(), item.Cantidad())
		}
		// El subtotal usa el precio guardado en la linea, no el precio actual
		// del catalogo, para respetar el precio del momento de la venta.
		subtotal += item.Subtotal()
	}

	// --- PASO 3: descuento escalonado segun el monto ---
	// Los tramos se revisan de mayor a menor. Si estuvieran al reves, un pedido
	// de $800 entraria en el primer tramo y recibiria 5% en vez de 15%.
	descuento := 0.0
	if subtotal >= montoParaQuincePorciento {
		descuento = 15.0
	} else if subtotal >= montoParaDiezPorciento {
		descuento = 10.0
	} else if subtotal >= montoParaCincoPorciento {
		descuento = 5.0
	}

	// --- PASO 4: calcular el total final ---
	// Se calcula primero cuanto dinero es el descuento y despues se resta, para
	// que el descuento mostrado y el total siempre coincidan.
	montoDescuento := subtotal * descuento / 100.0
	total := subtotal - montoDescuento

	// El pedido guarda el resultado; el texto se devuelve para mostrarlo.
	pedido.GuardarTotales(subtotal, descuento, total)

	if descuento > 0 {
		return fmt.Sprintf("Descuento del %.0f%% aplicado (-$%.2f)",
			descuento, montoDescuento), nil
	}
	return fmt.Sprintf("Sin descuento (se necesitan al menos $%.2f)",
		montoParaCincoPorciento), nil
}

// verTotalPedido calcula el total del pedido y muestra el detalle.
func verTotalPedido() {
	pedido, err := buscarPedido(leerTexto("Codigo del pedido: "))
	if err != nil {
		mostrarError(err)
		return
	}
	explicacion, err := calcularTotal(pedido)
	if err != nil {
		mostrarError(err)
		return
	}
	mostrarDetalle(pedido, explicacion)
}

// confirmarPedido cierra el pedido, descuenta el inventario de cada producto
// vendido y muestra las alertas de stock bajo que se generaron.
func confirmarPedido() {
	pedido, err := buscarPedido(leerTexto("Codigo del pedido: "))
	if err != nil {
		mostrarError(err)
		return
	}
	// Se recalcula antes de confirmar: el inventario pudo cambiar.
	explicacion, err := calcularTotal(pedido)
	if err != nil {
		mostrarError(err)
		return
	}

	alertas := []*models.Producto{}
	for _, item := range pedido.Items() {
		producto, err := buscarProducto(item.ProductoID())
		if err != nil {
			mostrarError(err)
			return
		}
		// Aqui SI se descuenta el inventario de verdad.
		if err := producto.DescontarStock(item.Cantidad()); err != nil {
			mostrarError(err)
			return
		}
		if producto.StockBajo() {
			alertas = append(alertas, producto)
		}
	}
	if err := pedido.Confirmar(); err != nil {
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
	if len(pedidos) == 0 {
		fmt.Println("No hay pedidos registrados.")
		return
	}
	fmt.Println()
	for _, pedido := range pedidos {
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
	producto, err := buscarProducto(leerTexto("Codigo del producto: "))
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
	producto, err := buscarProducto(leerTexto("Codigo del producto: "))
	if err != nil {
		mostrarError(err)
		return
	}
	cantidad, err := leerEntero("Nuevo stock: ")
	if err != nil {
		mostrarError(err)
		return
	}
	if err := producto.CambiarStock(cantidad); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Stock actualizado:", producto.Describir())
}

// reponerStock suma unidades al inventario de un producto.
func reponerStock() {
	producto, err := buscarProducto(leerTexto("Codigo del producto: "))
	if err != nil {
		mostrarError(err)
		return
	}
	cantidad, err := leerEntero("Unidades a ingresar: ")
	if err != nil {
		mostrarError(err)
		return
	}
	if err := producto.AumentarStock(cantidad); err != nil {
		mostrarError(err)
		return
	}
	fmt.Println("Inventario repuesto:", producto.Describir())
}

// verAlertas muestra los productos que llegaron al limite minimo.
func verAlertas() {
	encontrados := 0
	for _, producto := range productos {
		if producto.StockBajo() {
			if encontrados == 0 {
				fmt.Println("\n*** PRODUCTOS CON STOCK BAJO ***")
			}
			fmt.Printf("  - %-22s stock:%3d  minimo:%3d\n",
				producto.Nombre(), producto.Stock(), producto.StockMinimo())
			encontrados++
		}
	}
	if encontrados == 0 {
		fmt.Println("No hay alertas de stock bajo.")
	}
}

// ==========================================================================
// DEMOSTRACION DE POLIMORFISMO
// ==========================================================================

// verTodoElSistema guarda productos, clientes y pedidos en una misma lista de
// tipo models.Entity y los imprime con un solo bucle.
//
// Aunque son tres tipos distintos, todos cumplen la interfaz Entity porque
// todos tienen los metodos ID() y Describir(). Go decide en el momento de la
// ejecucion cual version de Describir() corresponde a cada elemento segun su
// tipo real: eso es polimorfismo.
func verTodoElSistema() {
	lista := []models.Entity{}

	for _, producto := range productos {
		lista = append(lista, producto)
	}
	for _, cliente := range clientes {
		lista = append(lista, cliente)
	}
	for _, pedido := range pedidos {
		lista = append(lista, pedido)
	}

	if len(lista) == 0 {
		fmt.Println("Todavia no hay datos cargados.")
		return
	}

	fmt.Println("\n--- TODOS LOS REGISTROS DEL SISTEMA ---")
	// Un solo bucle imprime los tres tipos distintos.
	for _, entidad := range lista {
		fmt.Println(entidad.Describir())
	}
}
