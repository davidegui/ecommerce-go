// Package services contiene la logica de negocio del sistema.
//
// Aqui viven las listas donde se guardan los datos y todas las operaciones que
// se pueden hacer con ellos: registrar, modificar, eliminar, buscar y calcular.
//
// Este paquete existe para que la misma logica pueda usarse desde dos lugares
// distintos sin repetir codigo: el menu de consola y los servicios web. Ninguno
// de los dos sabe como estan guardados los datos, solo llaman a estas funciones.
package services

import (
	"fmt"
	"strings"

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
	return producto, storage.GuardarProductos(productos)
}

// BuscarProducto devuelve el producto que tenga ese codigo.
func BuscarProducto(id string) (*models.Producto, error) {
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
	producto, err := BuscarProducto(id)
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
	return producto, storage.GuardarProductos(productos)
}

// EliminarProducto quita un producto de la lista.
func EliminarProducto(id string) error {
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

// ListarProductos devuelve el catalogo completo.
func ListarProductos() []*models.Producto {
	return productos
}

// ConsultarDisponibles devuelve solo los productos que tienen stock.
func ConsultarDisponibles() []*models.Producto {
	disponibles := []*models.Producto{}
	for _, producto := range productos {
		if producto.Stock() > 0 {
			disponibles = append(disponibles, producto)
		}
	}
	return disponibles
}

// BuscarPorNombre devuelve los productos cuyo nombre contenga el texto dado.
// La comparacion se hace en minusculas para que no distinga mayusculas, y usa
// Contains para aceptar coincidencias parciales.
func BuscarPorNombre(texto string) []*models.Producto {
	texto = strings.ToLower(strings.TrimSpace(texto))
	encontrados := []*models.Producto{}
	for _, producto := range productos {
		if strings.Contains(strings.ToLower(producto.Nombre()), texto) {
			encontrados = append(encontrados, producto)
		}
	}
	return encontrados
}

// ==========================================================================
// INVENTARIO
// ==========================================================================

// ActualizarStock fija las existencias de un producto en un valor exacto.
func ActualizarStock(id string, stock int) (*models.Producto, error) {
	producto, err := BuscarProducto(id)
	if err != nil {
		return nil, err
	}
	if err := producto.CambiarStock(stock); err != nil {
		return nil, err
	}
	return producto, storage.GuardarProductos(productos)
}

// ReponerStock suma unidades al inventario de un producto.
func ReponerStock(id string, cantidad int) (*models.Producto, error) {
	producto, err := BuscarProducto(id)
	if err != nil {
		return nil, err
	}
	if err := producto.AumentarStock(cantidad); err != nil {
		return nil, err
	}
	return producto, storage.GuardarProductos(productos)
}

// AlertasStockBajo devuelve los productos que llegaron al limite minimo.
func AlertasStockBajo() []*models.Producto {
	alertas := []*models.Producto{}
	for _, producto := range productos {
		if producto.StockBajo() {
			alertas = append(alertas, producto)
		}
	}
	return alertas
}

// ==========================================================================
// CLIENTES
// ==========================================================================

// RegistrarCliente crea un cliente nuevo, lo agrega a la lista y lo guarda.
func RegistrarCliente(nombre, email, telefono string) (*models.Cliente, error) {
	contadorClientes++
	codigo := fmt.Sprintf("C%03d", contadorClientes)

	cliente, err := models.NuevoCliente(codigo, nombre, email, telefono)
	if err != nil {
		contadorClientes--
		return nil, err
	}
	clientes = append(clientes, cliente)
	return cliente, storage.GuardarClientes(clientes)
}

// BuscarCliente devuelve el cliente que tenga ese codigo.
func BuscarCliente(id string) (*models.Cliente, error) {
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
	cliente, err := BuscarCliente(id)
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
	return cliente, storage.GuardarClientes(clientes)
}

// EliminarCliente quita un cliente de la lista.
func EliminarCliente(id string) error {
	id = strings.ToUpper(strings.TrimSpace(id))
	for i, cliente := range clientes {
		if cliente.ID() == id {
			clientes = append(clientes[:i], clientes[i+1:]...)
			return storage.GuardarClientes(clientes)
		}
	}
	return fmt.Errorf("%w: %s", utils.ErrClienteNoEncontrado, id)
}

// ListarClientes devuelve todos los clientes registrados.
func ListarClientes() []*models.Cliente {
	return clientes
}

// ==========================================================================
// PEDIDOS
// ==========================================================================

// CrearPedido crea un pedido vacio para un cliente que exista.
func CrearPedido(clienteID string) (*models.Pedido, error) {
	cliente, err := BuscarCliente(clienteID)
	if err != nil {
		return nil, err
	}
	contadorPedidos++
	codigo := fmt.Sprintf("O%03d", contadorPedidos)

	pedido := models.NuevoPedido(codigo, cliente.ID())
	pedidos = append(pedidos, pedido)
	return pedido, storage.GuardarPedidos(pedidos)
}

// BuscarPedido devuelve el pedido que tenga ese codigo.
func BuscarPedido(id string) (*models.Pedido, error) {
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
	pedido, err := BuscarPedido(pedidoID)
	if err != nil {
		return nil, err
	}
	producto, err := BuscarProducto(productoID)
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
	if _, err := CalcularTotal(pedido); err != nil {
		return nil, err
	}
	return pedido, storage.GuardarPedidos(pedidos)
}

// CalcularTotal es la funcion principal del sistema.
//
// Hace tres cosas: revisa que haya stock de cada producto del pedido, suma el
// subtotal, y aplica un descuento escalonado segun el monto. Devuelve un texto
// explicando el descuento y guarda los totales dentro del pedido.
//
// Si algun producto no tiene stock suficiente devuelve error y no modifica
// nada: es preferible rechazar el pedido completo a dejarlo mal calculado.
func CalcularTotal(pedido *models.Pedido) (string, error) {
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
		producto, err := BuscarProducto(item.ProductoID())
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
	pedido, err := BuscarPedido(pedidoID)
	if err != nil {
		return nil, "", err
	}
	explicacion, err := CalcularTotal(pedido)
	if err != nil {
		return nil, "", err
	}
	if err := storage.GuardarPedidos(pedidos); err != nil {
		return nil, "", err
	}
	return pedido, explicacion, nil
}

// ConfirmarPedido cierra el pedido, descuenta el inventario de cada producto
// vendido y devuelve las alertas de stock bajo que se generaron por la venta.
func ConfirmarPedido(pedidoID string) (*models.Pedido, string, []*models.Producto, error) {
	pedido, err := BuscarPedido(pedidoID)
	if err != nil {
		return nil, "", nil, err
	}
	// Se recalcula antes de confirmar: el inventario pudo cambiar.
	explicacion, err := CalcularTotal(pedido)
	if err != nil {
		return nil, "", nil, err
	}

	alertas := []*models.Producto{}
	for _, item := range pedido.Items() {
		producto, err := BuscarProducto(item.ProductoID())
		if err != nil {
			return nil, "", nil, err
		}
		// Aqui SI se descuenta el inventario de verdad.
		if err := producto.DescontarStock(item.Cantidad()); err != nil {
			return nil, "", nil, err
		}
		if producto.StockBajo() {
			alertas = append(alertas, producto)
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
	return pedido, explicacion, alertas, nil
}

// ListarPedidos devuelve todos los pedidos registrados.
func ListarPedidos() []*models.Pedido {
	return pedidos
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
	return lista
}
