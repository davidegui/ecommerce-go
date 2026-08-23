// Package services contiene la logica de negocio del sistema.
//
// Aqui viven las listas donde se guardan los datos y todas las operaciones que
// se pueden hacer con ellos: registrar, modificar, eliminar, buscar y calcular.
//
// Este paquete existe para que la misma logica pueda usarse desde dos lugares
// distintos sin repetir codigo: el menu de consola y los servicios web. Ninguno
// de los dos sabe como estan guardados los datos, solo llaman a estas funciones.
//
// # CONCURRENCIA
//
// El servidor web atiende cada peticion en una goroutine distinta, o sea que
// varias peticiones pueden estar ejecutandose al mismo tiempo sobre las mismas
// listas de datos.
//
// Si dos peticiones modificaran una lista a la vez, las dos leerian el mismo
// estado, las dos escribirian encima, y una de las dos modificaciones se
// perderia sin ningun aviso. A ese problema se lo llama condicion de carrera.
//
// Para evitarlo, este paquete protege las listas con un candado (sync.Mutex).
// El candado garantiza que solo una goroutine a la vez pueda estar dentro de
// las funciones que leen o modifican los datos: si una segunda goroutine llega
// mientras la primera trabaja, se queda esperando su turno.
package services

import (
	"fmt"
	"strings"
	"sync"

	"ecommerce/models"
	"ecommerce/storage"
	"ecommerce/utils"
)

// Listas donde se guardan los datos mientras el programa esta en ejecucion.
// Son slices de punteros: guardan la direccion de cada objeto, de modo que al
// modificar uno se modifica el que esta en la lista y no una copia.
var (
	productos []*models.Producto
	clientes  []*models.Cliente
	pedidos   []*models.Pedido
)

// Contadores para generar los codigos P001, C001 y O001.
var (
	contadorProductos int
	contadorClientes  int
	contadorPedidos   int
)

// candado protege las listas y los contadores de accesos simultaneos.
//
// Antes de tocar los datos, cada funcion llama a candado.Lock(): si otra
// goroutine ya lo tiene tomado, se queda esperando su turno. Al terminar llama
// a candado.Unlock() para liberarlo.
//
// Se usa siempre con defer, asi el candado se libera aunque la funcion termine
// antes de tiempo por un error. Olvidar liberarlo dejaria el programa colgado
// para siempre esperando un turno que nunca llega.
var candado sync.Mutex

// Reglas de descuento de la empresa.
// Estan como constantes para que la regla de negocio este en un solo lugar y no
// como numeros sueltos dentro del calculo.
const (
	montoParaCincoPorciento  = 100.0
	montoParaDiezPorciento   = 300.0
	montoParaQuincePorciento = 700.0
)

// ==========================================================================
// CARGA Y GUARDADO
// ==========================================================================

// CargarDatos lee los tres archivos JSON al iniciar el programa.
//
// Ademas de llenar las listas, recalcula los contadores buscando el numero mas
// alto de cada tipo de codigo. Se usa el maximo y no la cantidad de elementos
// porque, si se elimino un registro intermedio, contar generaria un codigo que
// ya existe.
func CargarDatos() error {
	candado.Lock()
	defer candado.Unlock()

	var err error
	if productos, err = storage.CargarProductos(); err != nil {
		return err
	}
	if clientes, err = storage.CargarClientes(); err != nil {
		return err
	}
	if pedidos, err = storage.CargarPedidos(); err != nil {
		return err
	}
	for _, p := range productos {
		if n := numeroDeCodigo(p.ID()); n > contadorProductos {
			contadorProductos = n
		}
	}
	for _, c := range clientes {
		if n := numeroDeCodigo(c.ID()); n > contadorClientes {
			contadorClientes = n
		}
	}
	for _, o := range pedidos {
		if n := numeroDeCodigo(o.ID()); n > contadorPedidos {
			contadorPedidos = n
		}
	}
	return nil
}

// numeroDeCodigo extrae la parte numerica de un codigo como P001 o C012.
// Si el codigo no tiene el formato esperado devuelve cero, para que un dato
// escrito a mano no detenga el programa.
func numeroDeCodigo(codigo string) int {
	if len(codigo) < 2 {
		return 0
	}
	numero := 0
	if _, err := fmt.Sscanf(codigo[1:], "%d", &numero); err != nil {
		return 0
	}
	return numero
}

// ==========================================================================
// PRODUCTOS
// ==========================================================================

// RegistrarProducto crea un producto nuevo, lo agrega a la lista y lo guarda.
func RegistrarProducto(nombre string, precio float64, stock, stockMinimo int) (*models.Producto, error) {
	// Lock toma el candado en modo exclusivo: nadie mas puede leer ni escribir
	// hasta que esta funcion termine.
	candado.Lock()
	defer candado.Unlock()

	contadorProductos++
	codigo := fmt.Sprintf("P%03d", contadorProductos)

	// El constructor valida los datos. Si algo esta mal se devuelve el codigo
	// al contador para no dejar huecos en la numeracion.
	producto, err := models.NuevoProducto(codigo, nombre, precio, stock, stockMinimo)
	if err != nil {
		contadorProductos--
		return nil, err
	}
	// append agrega al final del slice y devuelve el slice nuevo, por eso hay
	// que reasignarlo a la lista.
	productos = append(productos, producto)
	// Se devuelve una copia y no el objeto original, para que quien la reciba
	// pueda leerla sin riesgo aunque otra goroutine modifique el catalogo.
	return producto.Copia(), storage.GuardarProductos(productos)
}

// BuscarProducto devuelve el producto que tenga ese codigo.
//
// Toma el candado en modo lectura (RLock) porque solo consulta la lista sin
// modificarla. Varias goroutines pueden ejecutar esta funcion al mismo tiempo.
func BuscarProducto(id string) (*models.Producto, error) {
	candado.Lock()
	defer candado.Unlock()

	producto, err := buscarProductoSinCandado(id)
	if err != nil {
		return nil, err
	}
	return producto.Copia(), nil
}

// buscarProductoSinCandado hace la busqueda sin tomar el candado.
//
// Existe porque Go no permite tomar dos veces el mismo candado: si una funcion
// que ya lo tomo llamara a BuscarProducto, el programa se quedaria trabado
// esperando un candado que el mismo tiene. Se le llama interbloqueo o deadlock.
// Por eso las funciones internas usan esta version y solo las publicas toman
// el candado.
func buscarProductoSinCandado(id string) (*models.Producto, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	// range recorre la lista; el indice se descarta con _ porque no se usa.
	for _, producto := range productos {
		if producto.ID() == id {
			return producto, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrProductoNoEncontrado, id)
}

// ModificarProducto cambia nombre, precio o stock minimo de un producto.
// Un nombre vacio, un precio en cero o un stock minimo negativo significan que
// ese dato no se debe modificar.
func ModificarProducto(id, nombre string, precio float64, stockMinimo int) (*models.Producto, error) {
	candado.Lock()
	defer candado.Unlock()

	producto, err := buscarProductoSinCandado(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(nombre) != "" {
		if err := producto.CambiarNombre(nombre); err != nil {
			return nil, err
		}
	}
	if precio > 0 {
		if err := producto.CambiarPrecio(precio); err != nil {
			return nil, err
		}
	}
	if stockMinimo >= 0 {
		if err := producto.CambiarStockMinimo(stockMinimo); err != nil {
			return nil, err
		}
	}
	return producto.Copia(), storage.GuardarProductos(productos)
}

// EliminarProducto quita un producto de la lista.
func EliminarProducto(id string) error {
	candado.Lock()
	defer candado.Unlock()

	id = strings.ToUpper(strings.TrimSpace(id))
	for i, producto := range productos {
		if producto.ID() == id {
			// Para borrar de un slice se pegan las dos partes que quedan: lo que
			// esta antes del elemento y lo que esta despues. Los tres puntos
			// desarman el segundo slice en elementos sueltos.
			productos = append(productos[:i], productos[i+1:]...)
			return storage.GuardarProductos(productos)
		}
	}
	return fmt.Errorf("%w: %s", utils.ErrProductoNoEncontrado, id)
}

// ListarProductos devuelve una copia del catalogo completo.
//
// Se devuelve una copia del slice y no el original porque, si otra goroutine
// agrega o elimina un producto mientras quien recibe la lista la esta
// recorriendo, el recorrido podria fallar. La copia es independiente.
func ListarProductos() []*models.Producto {
	candado.Lock()
	defer candado.Unlock()

	copia := make([]*models.Producto, 0, len(productos))
	for _, producto := range productos {
		copia = append(copia, producto.Copia())
	}
	return copia
}

// ConsultarDisponibles devuelve solo los productos que tienen stock.
func ConsultarDisponibles() []*models.Producto {
	candado.Lock()
	defer candado.Unlock()

	disponibles := []*models.Producto{}
	for _, producto := range productos {
		if producto.Stock() > 0 {
			disponibles = append(disponibles, producto.Copia())
		}
	}
	return disponibles
}

// BuscarPorNombre devuelve los productos cuyo nombre contenga el texto dado.
// La comparacion se hace en minusculas para que no distinga mayusculas, y usa
// Contains para aceptar coincidencias parciales.
func BuscarPorNombre(texto string) []*models.Producto {
	candado.Lock()
	defer candado.Unlock()

	texto = strings.ToLower(strings.TrimSpace(texto))
	encontrados := []*models.Producto{}
	for _, producto := range productos {
		if strings.Contains(strings.ToLower(producto.Nombre()), texto) {
			encontrados = append(encontrados, producto.Copia())
		}
	}
	return encontrados
}

// ==========================================================================
// INVENTARIO
// ==========================================================================

// ActualizarStock fija las existencias de un producto en un valor exacto.
func ActualizarStock(id string, stock int) (*models.Producto, error) {
	candado.Lock()
	defer candado.Unlock()

	producto, err := buscarProductoSinCandado(id)
	if err != nil {
		return nil, err
	}
	if err := producto.CambiarStock(stock); err != nil {
		return nil, err
	}
	return producto.Copia(), storage.GuardarProductos(productos)
}

// ReponerStock suma unidades al inventario de un producto.
func ReponerStock(id string, cantidad int) (*models.Producto, error) {
	candado.Lock()
	defer candado.Unlock()

	producto, err := buscarProductoSinCandado(id)
	if err != nil {
		return nil, err
	}
	if err := producto.AumentarStock(cantidad); err != nil {
		return nil, err
	}
	return producto.Copia(), storage.GuardarProductos(productos)
}

// AlertasStockBajo devuelve los productos que llegaron al limite minimo.
func AlertasStockBajo() []*models.Producto {
	candado.Lock()
	defer candado.Unlock()

	alertas := []*models.Producto{}
	for _, producto := range productos {
		if producto.StockBajo() {
			alertas = append(alertas, producto.Copia())
		}
	}
	return alertas
}

// ==========================================================================
// CLIENTES
// ==========================================================================

// RegistrarCliente crea un cliente nuevo, lo agrega a la lista y lo guarda.
func RegistrarCliente(nombre, email, telefono string) (*models.Cliente, error) {
	candado.Lock()
	defer candado.Unlock()

	contadorClientes++
	codigo := fmt.Sprintf("C%03d", contadorClientes)

	cliente, err := models.NuevoCliente(codigo, nombre, email, telefono)
	if err != nil {
		contadorClientes--
		return nil, err
	}
	clientes = append(clientes, cliente)
	return cliente.Copia(), storage.GuardarClientes(clientes)
}

// BuscarCliente devuelve el cliente que tenga ese codigo.
// Toma el candado en modo lectura porque solo consulta.
func BuscarCliente(id string) (*models.Cliente, error) {
	candado.Lock()
	defer candado.Unlock()

	cliente, err := buscarClienteSinCandado(id)
	if err != nil {
		return nil, err
	}
	return cliente.Copia(), nil
}

// buscarClienteSinCandado busca sin tomar el candado, para usarse desde
// funciones que ya lo tomaron.
func buscarClienteSinCandado(id string) (*models.Cliente, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, cliente := range clientes {
		if cliente.ID() == id {
			return cliente, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrClienteNoEncontrado, id)
}

// ActualizarCliente cambia los datos de un cliente ya registrado.
// Los campos vacios se dejan como estaban.
func ActualizarCliente(id, nombre, email, telefono string) (*models.Cliente, error) {
	candado.Lock()
	defer candado.Unlock()

	cliente, err := buscarClienteSinCandado(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(nombre) != "" {
		if err := cliente.CambiarNombre(nombre); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(email) != "" {
		if err := cliente.CambiarEmail(email); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(telefono) != "" {
		if err := cliente.CambiarTelefono(telefono); err != nil {
			return nil, err
		}
	}
	return cliente.Copia(), storage.GuardarClientes(clientes)
}

// EliminarCliente quita un cliente de la lista.
func EliminarCliente(id string) error {
	candado.Lock()
	defer candado.Unlock()

	id = strings.ToUpper(strings.TrimSpace(id))
	for i, cliente := range clientes {
		if cliente.ID() == id {
			clientes = append(clientes[:i], clientes[i+1:]...)
			return storage.GuardarClientes(clientes)
		}
	}
	return fmt.Errorf("%w: %s", utils.ErrClienteNoEncontrado, id)
}

// ListarClientes devuelve una copia de la lista de clientes.
func ListarClientes() []*models.Cliente {
	candado.Lock()
	defer candado.Unlock()

	copia := make([]*models.Cliente, 0, len(clientes))
	for _, cliente := range clientes {
		copia = append(copia, cliente.Copia())
	}
	return copia
}

// ==========================================================================
// PEDIDOS
// ==========================================================================

// CrearPedido crea un pedido vacio para un cliente que exista.
func CrearPedido(clienteID string) (*models.Pedido, error) {
	candado.Lock()
	defer candado.Unlock()

	cliente, err := buscarClienteSinCandado(clienteID)
	if err != nil {
		return nil, err
	}
	contadorPedidos++
	codigo := fmt.Sprintf("O%03d", contadorPedidos)

	pedido := models.NuevoPedido(codigo, cliente.ID())
	pedidos = append(pedidos, pedido)
	return pedido.Copia(), storage.GuardarPedidos(pedidos)
}

// BuscarPedido devuelve el pedido que tenga ese codigo.
// Toma el candado en modo lectura porque solo consulta.
func BuscarPedido(id string) (*models.Pedido, error) {
	candado.Lock()
	defer candado.Unlock()

	pedido, err := buscarPedidoSinCandado(id)
	if err != nil {
		return nil, err
	}
	return pedido.Copia(), nil
}

// buscarPedidoSinCandado busca sin tomar el candado, para usarse desde
// funciones que ya lo tomaron.
func buscarPedidoSinCandado(id string) (*models.Pedido, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, pedido := range pedidos {
		if pedido.ID() == id {
			return pedido, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", utils.ErrPedidoNoEncontrado, id)
}

// AgregarProductoAPedido agrega una linea al detalle del pedido.
//
// Revisa que haya stock suficiente, pero NO descuenta inventario todavia: el
// descuento real ocurre unicamente al confirmar el pedido.
func AgregarProductoAPedido(pedidoID, productoID string, cantidad int) (*models.Pedido, error) {
	// Se toma el candado exclusivo durante toda la operacion: verificar el
	// stock y agregar la linea deben ocurrir juntos. Si otra goroutine vendiera
	// el ultimo producto entre esas dos acciones, el pedido quedaria con una
	// cantidad que el inventario ya no puede cubrir.
	candado.Lock()
	defer candado.Unlock()

	pedido, err := buscarPedidoSinCandado(pedidoID)
	if err != nil {
		return nil, err
	}
	producto, err := buscarProductoSinCandado(productoID)
	if err != nil {
		return nil, err
	}
	if !producto.HayStock(cantidad) {
		return nil, fmt.Errorf("%w: %s (hay %d, se pidieron %d)",
			utils.ErrStockInsuficiente, producto.Nombre(), producto.Stock(), cantidad)
	}
	item, err := models.NuevoItem(producto.ID(), producto.Nombre(), producto.Precio(), cantidad)
	if err != nil {
		return nil, err
	}
	if err := pedido.AgregarItem(item); err != nil {
		return nil, err
	}
	// Se recalcula el total para que el pedido quede siempre coherente.
	if _, err := calcularTotalSinCandado(pedido); err != nil {
		return nil, err
	}
	return pedido.Copia(), storage.GuardarPedidos(pedidos)
}

// CalcularTotal es la funcion principal del sistema.
//
// Nota sobre concurrencia: esta funcion NO toma el candado. Siempre se llama
// desde funciones que ya lo tomaron, porque el calculo debe ocurrir dentro de
// la misma operacion indivisible que agrega la linea o confirma el pedido.
//
// Hace tres cosas: revisa que haya stock de cada producto del pedido, suma el
// subtotal, y aplica un descuento escalonado segun el monto. Devuelve un texto
// explicando el descuento y guarda los totales dentro del pedido.
//
// Si algun producto no tiene stock suficiente devuelve error y no modifica
// nada: es preferible rechazar el pedido completo a dejarlo mal calculado.
func CalcularTotal(pedido *models.Pedido) (string, error) {
	candado.Lock()
	defer candado.Unlock()
	return calcularTotalSinCandado(pedido)
}

// calcularTotalSinCandado hace el calculo sin tomar el candado, para que la
// usen las funciones que ya lo tienen tomado.
func calcularTotalSinCandado(pedido *models.Pedido) (string, error) {
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
		producto, err := buscarProductoSinCandado(item.ProductoID())
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
	// Se redondea una sola vez, al final de la suma, para no arrastrar los
	// errores de representacion decimal de los float64.
	subtotal = utils.RedondearDinero(subtotal)

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
	montoDescuento := utils.RedondearDinero(subtotal * descuento / 100.0)
	total := utils.RedondearDinero(subtotal - montoDescuento)

	// El pedido guarda el resultado; el texto se devuelve para mostrarlo.
	pedido.GuardarTotales(subtotal, descuento, total)

	if descuento > 0 {
		return fmt.Sprintf("Descuento del %.0f%% aplicado (-$%.2f)",
			descuento, montoDescuento), nil
	}
	return fmt.Sprintf("Sin descuento (se necesitan al menos $%.2f)",
		montoParaCincoPorciento), nil
}

// AplicarDescuento recalcula el total de un pedido guardado y lo devuelve.
func AplicarDescuento(pedidoID string) (*models.Pedido, string, error) {
	candado.Lock()
	defer candado.Unlock()

	pedido, err := buscarPedidoSinCandado(pedidoID)
	if err != nil {
		return nil, "", err
	}
	// Version sin candado: esta funcion ya lo tiene tomado.
	explicacion, err := calcularTotalSinCandado(pedido)
	if err != nil {
		return nil, "", err
	}
	if err := storage.GuardarPedidos(pedidos); err != nil {
		return nil, "", err
	}
	return pedido.Copia(), explicacion, nil
}

// ConfirmarPedido cierra el pedido, descuenta el inventario de cada producto
// vendido y devuelve las alertas de stock bajo que se generaron por la venta.
func ConfirmarPedido(pedidoID string) (*models.Pedido, string, []*models.Producto, error) {
	// Toda la confirmacion ocurre bajo un solo candado exclusivo: recalcular el
	// total, descontar el inventario y cerrar el pedido tienen que ser una sola
	// operacion indivisible. Si dos personas confirmaran pedidos del mismo
	// producto al mismo tiempo, ambas podrian pasar la validacion de stock y el
	// inventario terminaria en negativo.
	candado.Lock()
	defer candado.Unlock()

	pedido, err := buscarPedidoSinCandado(pedidoID)
	if err != nil {
		return nil, "", nil, err
	}
	// Se recalcula antes de confirmar: el inventario pudo cambiar.
	// Se usa la version sin candado porque esta funcion ya lo tiene tomado.
	explicacion, err := calcularTotalSinCandado(pedido)
	if err != nil {
		return nil, "", nil, err
	}

	alertas := []*models.Producto{}
	for _, item := range pedido.Items() {
		producto, err := buscarProductoSinCandado(item.ProductoID())
		if err != nil {
			return nil, "", nil, err
		}
		// Aqui SI se descuenta el inventario de verdad.
		if err := producto.DescontarStock(item.Cantidad()); err != nil {
			return nil, "", nil, err
		}
		if producto.StockBajo() {
			alertas = append(alertas, producto.Copia())
		}
	}
	if err := pedido.Confirmar(); err != nil {
		return nil, "", nil, err
	}
	if err := storage.GuardarProductos(productos); err != nil {
		return nil, "", nil, err
	}
	if err := storage.GuardarPedidos(pedidos); err != nil {
		return nil, "", nil, err
	}
	return pedido.Copia(), explicacion, alertas, nil
}

// ListarPedidos devuelve una copia de la lista de pedidos.
func ListarPedidos() []*models.Pedido {
	candado.Lock()
	defer candado.Unlock()

	copia := make([]*models.Pedido, 0, len(pedidos))
	for _, pedido := range pedidos {
		copia = append(copia, pedido.Copia())
	}
	return copia
}

// ==========================================================================
// POLIMORFISMO
// ==========================================================================

// TodasLasEntidades devuelve productos, clientes y pedidos en una sola lista.
//
// Los tres tipos son distintos, pero todos cumplen la interfaz models.Entity
// porque todos tienen los metodos ID() y Describir(). Por eso pueden guardarse
// juntos y recorrerse con un solo bucle: eso es polimorfismo.
func TodasLasEntidades() []models.Entity {
	candado.Lock()
	defer candado.Unlock()

	lista := []models.Entity{}
	for _, producto := range productos {
		lista = append(lista, producto.Copia())
	}
	for _, cliente := range clientes {
		lista = append(lista, cliente.Copia())
	}
	for _, pedido := range pedidos {
		lista = append(lista, pedido.Copia())
	}
	return lista
}
