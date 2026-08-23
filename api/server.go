// Package api expone las funcionalidades del sistema como servicios web.
//
// Un servicio web es una funcion del programa que se puede llamar desde fuera
// usando el protocolo HTTP, en lugar de usar el menu de consola. El cliente
// (un navegador, Postman, o cualquier otro programa) envia una peticion a una
// direccion, y el servidor responde con datos en formato JSON.
//
// Todo esto se hace con el paquete net/http de la libreria estandar de Go, sin
// usar ningun framework externo.
//
// La logica de negocio no se repite aqui: cada handler llama a las mismas
// funciones del paquete services que usa el menu de consola.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ecommerce/services"
	"ecommerce/utils"
)

// ==========================================================================
// ESTRUCTURAS DE RESPUESTA
// ==========================================================================

// Respuesta es el formato uniforme de todas las respuestas del servidor.
//
// Tener siempre la misma estructura facilita el trabajo de quien consume el
// servicio: sabe que siempre va a recibir un campo "exito", y despues o bien
// "datos" o bien "error", nunca los dos.
//
// El tag omitempty le dice a encoding/json que omita el campo cuando este
// vacio, para que la respuesta no tenga campos nulos innecesarios.
type Respuesta struct {
	Exito   bool        `json:"exito"`
	Mensaje string      `json:"mensaje,omitempty"`
	Datos   interface{} `json:"datos,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// responderJSON escribe una respuesta en formato JSON con el codigo indicado.
//
// El encabezado Content-Type le avisa al cliente que el cuerpo de la respuesta
// es JSON. El codigo de estado indica si la peticion salio bien (200), si se
// creo algo (201), si los datos estaban mal (400) o si no se encontro lo que se
// pedia (404).
func responderJSON(w http.ResponseWriter, codigo int, respuesta Respuesta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	// El encoder convierte el struct a JSON y lo escribe directamente en la
	// respuesta. SetIndent hace que salga formateado y legible.
	codificador := json.NewEncoder(w)
	codificador.SetIndent("", "  ")
	if err := codificador.Encode(respuesta); err != nil {
		fmt.Println("error al enviar la respuesta:", err)
	}
}

// responderError envia un error traduciendolo al codigo HTTP que corresponde.
//
// Aqui se aprovecha el manejo de errores del sistema: con errors.Is se
// reconoce que tipo de error ocurrio, aunque venga envuelto con informacion
// extra, y segun eso se elige el codigo HTTP correcto. Un producto que no
// existe es un 404, un dato mal escrito es un 400.
func responderError(w http.ResponseWriter, err error) {
	codigo := http.StatusBadRequest // 400 por defecto: datos incorrectos

	switch {
	case errors.Is(err, utils.ErrProductoNoEncontrado),
		errors.Is(err, utils.ErrClienteNoEncontrado),
		errors.Is(err, utils.ErrPedidoNoEncontrado):
		codigo = http.StatusNotFound // 404: no existe
	case errors.Is(err, utils.ErrStockInsuficiente):
		codigo = http.StatusConflict // 409: existe pero no se puede hacer
	case errors.Is(err, utils.ErrArchivo), errors.Is(err, utils.ErrArchivoInvalido):
		codigo = http.StatusInternalServerError // 500: fallo del servidor
	}

	responderJSON(w, codigo, Respuesta{
		Exito: false,
		Error: err.Error(),
	})
}

// leerCuerpo convierte el cuerpo JSON de la peticion en el struct indicado.
// El destino se recibe como interface{} para que la misma funcion sirva con
// cualquier tipo de peticion.
func leerCuerpo(r *http.Request, destino interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(destino); err != nil {
		return fmt.Errorf("el cuerpo de la peticion no es un JSON valido")
	}
	return nil
}

// ==========================================================================
// SERVICIO 1: REGISTRAR PRODUCTO   ->   POST /api/productos
// ==========================================================================

// peticionProducto es el JSON que se espera recibir al registrar o modificar
// un producto.
type peticionProducto struct {
	Nombre      string  `json:"nombre"`
	Precio      float64 `json:"precio"`
	Stock       int     `json:"stock"`
	StockMinimo int     `json:"stock_minimo"`
}

// registrarProducto atiende POST /api/productos.
// Recibe los datos del producto en el cuerpo de la peticion y lo registra.
func registrarProducto(w http.ResponseWriter, r *http.Request) {
	var peticion peticionProducto
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	producto, err := services.RegistrarProducto(
		peticion.Nombre, peticion.Precio, peticion.Stock, peticion.StockMinimo)
	if err != nil {
		responderError(w, err)
		return
	}
	// 201 Created indica que se creo un recurso nuevo.
	responderJSON(w, http.StatusCreated, Respuesta{
		Exito:   true,
		Mensaje: "Producto registrado correctamente",
		Datos:   producto,
	})
}

// ==========================================================================
// SERVICIO 2: CONSULTAR PRODUCTOS   ->   GET /api/productos
// ==========================================================================

// listarProductos atiende GET /api/productos.
//
// Acepta dos filtros opcionales en la direccion:
//
//	/api/productos?disponibles=true   devuelve solo los que tienen stock
//	/api/productos?nombre=teclado     busca por coincidencia parcial de nombre
func listarProductos(w http.ResponseWriter, r *http.Request) {
	// URL.Query() lee los parametros que van despues del signo de pregunta.
	filtroNombre := r.URL.Query().Get("nombre")
	soloDisponibles := r.URL.Query().Get("disponibles") == "true"

	lista := services.ListarProductos()
	if strings.TrimSpace(filtroNombre) != "" {
		lista = services.BuscarPorNombre(filtroNombre)
	} else if soloDisponibles {
		lista = services.ConsultarDisponibles()
	}

	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("%d productos encontrados", len(lista)),
		Datos:   lista,
	})
}

// ==========================================================================
// SERVICIO 3: MODIFICAR PRODUCTO   ->   PUT /api/productos/{id}
// ==========================================================================

// modificarProducto atiende PUT /api/productos/{id}.
// El codigo del producto viaja en la direccion y los datos nuevos en el cuerpo.
func modificarProducto(w http.ResponseWriter, r *http.Request) {
	// PathValue lee la parte variable de la ruta, o sea el {id}.
	id := r.PathValue("id")

	var peticion peticionProducto
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	// Un stock minimo de cero es un valor valido, pero el servicio interpreta
	// los negativos como "no cambiar este dato". Como el JSON no distingue
	// entre "cero" y "no enviado", se usa -1 cuando no viene el campo.
	stockMinimo := peticion.StockMinimo
	if stockMinimo == 0 {
		stockMinimo = -1
	}

	producto, err := services.ModificarProducto(id, peticion.Nombre, peticion.Precio, stockMinimo)
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: "Producto modificado correctamente",
		Datos:   producto,
	})
}

// ==========================================================================
// SERVICIO 4: ELIMINAR PRODUCTO   ->   DELETE /api/productos/{id}
// ==========================================================================

// eliminarProducto atiende DELETE /api/productos/{id}.
func eliminarProducto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := services.EliminarProducto(id); err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("Producto %s eliminado", strings.ToUpper(id)),
	})
}

// ==========================================================================
// SERVICIO 5: REGISTRAR CLIENTE   ->   POST /api/clientes
// ==========================================================================

// peticionCliente es el JSON que se espera al registrar o actualizar un cliente.
type peticionCliente struct {
	Nombre   string `json:"nombre"`
	Email    string `json:"correo"`
	Telefono string `json:"telefono"`
}

// registrarCliente atiende POST /api/clientes.
func registrarCliente(w http.ResponseWriter, r *http.Request) {
	var peticion peticionCliente
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	cliente, err := services.RegistrarCliente(peticion.Nombre, peticion.Email, peticion.Telefono)
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, Respuesta{
		Exito:   true,
		Mensaje: "Cliente registrado correctamente",
		Datos:   cliente,
	})
}

// ==========================================================================
// SERVICIO 6: CONSULTAR CLIENTES   ->   GET /api/clientes
// ==========================================================================

// listarClientes atiende GET /api/clientes.
func listarClientes(w http.ResponseWriter, r *http.Request) {
	lista := services.ListarClientes()
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("%d clientes registrados", len(lista)),
		Datos:   lista,
	})
}

// ==========================================================================
// SERVICIO 7: ACTUALIZAR CLIENTE   ->   PUT /api/clientes/{id}
// ==========================================================================

// actualizarCliente atiende PUT /api/clientes/{id}.
func actualizarCliente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var peticion peticionCliente
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	cliente, err := services.ActualizarCliente(id, peticion.Nombre, peticion.Email, peticion.Telefono)
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: "Cliente actualizado correctamente",
		Datos:   cliente,
	})
}

// ==========================================================================
// SERVICIO 8: CREAR PEDIDO   ->   POST /api/pedidos
// ==========================================================================

// peticionPedido es el JSON que se espera al crear un pedido.
type peticionPedido struct {
	ClienteID string `json:"cliente_id"`
}

// crearPedido atiende POST /api/pedidos.
func crearPedido(w http.ResponseWriter, r *http.Request) {
	var peticion peticionPedido
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	pedido, err := services.CrearPedido(peticion.ClienteID)
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, Respuesta{
		Exito:   true,
		Mensaje: "Pedido creado correctamente",
		Datos:   pedido,
	})
}

// ==========================================================================
// SERVICIO 9: AGREGAR PRODUCTO AL PEDIDO
//             ->   POST /api/pedidos/{id}/productos
// ==========================================================================

// peticionItem es el JSON que se espera al agregar un producto a un pedido.
type peticionItem struct {
	ProductoID string `json:"producto_id"`
	Cantidad   int    `json:"cantidad"`
}

// agregarProductoAPedido atiende POST /api/pedidos/{id}/productos.
// Verifica que haya stock, pero no descuenta inventario: eso ocurre al
// confirmar el pedido.
func agregarProductoAPedido(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var peticion peticionItem
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}
	pedido, err := services.AgregarProductoAPedido(id, peticion.ProductoID, peticion.Cantidad)
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: "Producto agregado al pedido",
		Datos:   pedido,
	})
}

// ==========================================================================
// SERVICIO 10: CONFIRMAR PEDIDO   ->   POST /api/pedidos/{id}/confirmar
// ==========================================================================

// resultadoConfirmacion es el JSON que se devuelve al confirmar un pedido.
// Ademas del pedido incluye las alertas de stock bajo generadas por la venta.
type resultadoConfirmacion struct {
	Pedido    interface{} `json:"pedido"`
	Descuento string      `json:"descuento_aplicado"`
	Alertas   []string    `json:"alertas_stock_bajo"`
}

// confirmarPedido atiende POST /api/pedidos/{id}/confirmar.
// Recalcula el total, descuenta el inventario y cierra el pedido.
func confirmarPedido(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	pedido, explicacion, productosEnAlerta, err := services.ConfirmarPedido(id)
	if err != nil {
		responderError(w, err)
		return
	}
	// Las alertas se convierten a texto legible antes de enviarlas.
	alertas := []string{}
	for _, producto := range productosEnAlerta {
		alertas = append(alertas, fmt.Sprintf("%s: quedan %d unidades (minimo %d)",
			producto.Nombre(), producto.Stock(), producto.StockMinimo()))
	}

	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: "Pedido confirmado e inventario descontado",
		Datos: resultadoConfirmacion{
			Pedido:    pedido,
			Descuento: explicacion,
			Alertas:   alertas,
		},
	})
}

// ==========================================================================
// SERVICIO 11: CONSULTAR PEDIDOS   ->   GET /api/pedidos
// ==========================================================================

// listarPedidos atiende GET /api/pedidos.
func listarPedidos(w http.ResponseWriter, r *http.Request) {
	lista := services.ListarPedidos()
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("%d pedidos registrados", len(lista)),
		Datos:   lista,
	})
}

// ==========================================================================
// SERVICIO 12: INVENTARIO   ->   GET y PUT /api/inventario
// ==========================================================================

// peticionInventario es el JSON que se espera al actualizar el inventario.
// La operacion puede ser "fijar" (poner un valor exacto) o "reponer" (sumar).
type peticionInventario struct {
	ProductoID string `json:"producto_id"`
	Cantidad   int    `json:"cantidad"`
	Operacion  string `json:"operacion"`
}

// consultarInventario atiende GET /api/inventario.
// Sin parametros devuelve las alertas de stock bajo.
// Con ?producto=P001 devuelve el estado de inventario de ese producto.
func consultarInventario(w http.ResponseWriter, r *http.Request) {
	productoID := r.URL.Query().Get("producto")

	if strings.TrimSpace(productoID) != "" {
		producto, err := services.BuscarProducto(productoID)
		if err != nil {
			responderError(w, err)
			return
		}
		responderJSON(w, http.StatusOK, Respuesta{
			Exito:   true,
			Mensaje: producto.Describir(),
			Datos:   producto,
		})
		return
	}

	alertas := services.AlertasStockBajo()
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("%d productos con stock bajo", len(alertas)),
		Datos:   alertas,
	})
}

// actualizarInventario atiende PUT /api/inventario.
// Segun el campo "operacion" fija el stock en un valor exacto o suma unidades.
func actualizarInventario(w http.ResponseWriter, r *http.Request) {
	var peticion peticionInventario
	if err := leerCuerpo(r, &peticion); err != nil {
		responderError(w, err)
		return
	}

	var producto interface{}
	var err error

	switch strings.ToLower(strings.TrimSpace(peticion.Operacion)) {
	case "reponer":
		producto, err = services.ReponerStock(peticion.ProductoID, peticion.Cantidad)
	case "fijar", "":
		producto, err = services.ActualizarStock(peticion.ProductoID, peticion.Cantidad)
	default:
		responderError(w, fmt.Errorf("operacion '%s' no valida: use 'fijar' o 'reponer'",
			peticion.Operacion))
		return
	}
	if err != nil {
		responderError(w, err)
		return
	}
	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: "Inventario actualizado",
		Datos:   producto,
	})
}

// ==========================================================================
// SERVICIO 13: RESUMEN DEL SISTEMA (POLIMORFISMO)  ->  GET /api/resumen
// ==========================================================================

// resumenSistema atiende GET /api/resumen.
//
// Devuelve la descripcion de todos los registros del sistema recorriendo una
// sola lista de tipo models.Entity que contiene productos, clientes y pedidos
// mezclados. Cada elemento ejecuta su propia version del metodo Describir, y Go
// decide cual segun el tipo real de cada uno: eso es polimorfismo.
func resumenSistema(w http.ResponseWriter, r *http.Request) {
	entidades := services.TodasLasEntidades()

	descripciones := []string{}
	for _, entidad := range entidades {
		descripciones = append(descripciones, entidad.Describir())
	}

	responderJSON(w, http.StatusOK, Respuesta{
		Exito:   true,
		Mensaje: fmt.Sprintf("%d registros en el sistema", len(descripciones)),
		Datos:   descripciones,
	})
}

// ==========================================================================
// REGISTRO DE RUTAS Y ARRANQUE DEL SERVIDOR
// ==========================================================================

// Iniciar levanta el servidor web en el puerto indicado.
//
// ServeMux es el enrutador de la libreria estandar: asocia cada direccion con
// la funcion que debe atenderla. Desde Go 1.22 se puede indicar el metodo HTTP
// y partes variables de la ruta con llaves, por ejemplo {id}.
func Iniciar(puerto int) error {
	mux := http.NewServeMux()

	// Productos
	mux.HandleFunc("POST /api/productos", registrarProducto)
	mux.HandleFunc("GET /api/productos", listarProductos)
	mux.HandleFunc("PUT /api/productos/{id}", modificarProducto)
	mux.HandleFunc("DELETE /api/productos/{id}", eliminarProducto)

	// Clientes
	mux.HandleFunc("POST /api/clientes", registrarCliente)
	mux.HandleFunc("GET /api/clientes", listarClientes)
	mux.HandleFunc("PUT /api/clientes/{id}", actualizarCliente)

	// Pedidos
	mux.HandleFunc("POST /api/pedidos", crearPedido)
	mux.HandleFunc("GET /api/pedidos", listarPedidos)
	mux.HandleFunc("POST /api/pedidos/{id}/productos", agregarProductoAPedido)
	mux.HandleFunc("POST /api/pedidos/{id}/confirmar", confirmarPedido)

	// Inventario
	mux.HandleFunc("GET /api/inventario", consultarInventario)
	mux.HandleFunc("PUT /api/inventario", actualizarInventario)

	// Resumen general
	mux.HandleFunc("GET /api/resumen", resumenSistema)

	direccion := ":" + strconv.Itoa(puerto)

	fmt.Println("\n==========================================================")
	fmt.Println("   SERVIDOR WEB INICIADO")
	fmt.Println("==========================================================")
	fmt.Printf("   Direccion base: http://localhost%s\n\n", direccion)
	fmt.Println("   PRODUCTOS")
	fmt.Println("     POST   /api/productos              registrar producto")
	fmt.Println("     GET    /api/productos              consultar productos")
	fmt.Println("     PUT    /api/productos/{id}         modificar producto")
	fmt.Println("     DELETE /api/productos/{id}         eliminar producto")
	fmt.Println("   CLIENTES")
	fmt.Println("     POST   /api/clientes               registrar cliente")
	fmt.Println("     GET    /api/clientes               consultar clientes")
	fmt.Println("     PUT    /api/clientes/{id}          actualizar cliente")
	fmt.Println("   PEDIDOS")
	fmt.Println("     POST   /api/pedidos                crear pedido")
	fmt.Println("     GET    /api/pedidos                consultar pedidos")
	fmt.Println("     POST   /api/pedidos/{id}/productos agregar producto")
	fmt.Println("     POST   /api/pedidos/{id}/confirmar confirmar pedido")
	fmt.Println("   INVENTARIO")
	fmt.Println("     GET    /api/inventario             consultar stock")
	fmt.Println("     PUT    /api/inventario             actualizar stock")
	fmt.Println("   GENERAL")
	fmt.Println("     GET    /api/resumen                resumen del sistema")
	fmt.Println("\n   Presione Ctrl+C para detener el servidor.")
	fmt.Println("==========================================================")

	// ListenAndServe se queda escuchando peticiones y no devuelve el control
	// hasta que el servidor se detiene o falla.
	return http.ListenAndServe(direccion, mux)
}
