package models

import (
	"encoding/json"
	"fmt"

	"ecommerce/utils"
)

// Estados posibles de un pedido.
const (
	EstadoPendiente  = "PENDIENTE"
	EstadoConfirmado = "CONFIRMADO"
)

// ItemPedido es una linea del detalle del pedido.
// Guarda una COPIA del nombre y del precio del producto: si manana cambia el
// precio en el catalogo, este pedido debe seguir mostrando el precio con el
// que se vendio.
type ItemPedido struct {
	productoID string
	nombre     string
	precio     float64
	cantidad   int
}

// NuevoItem construye una linea de detalle.
func NuevoItem(productoID, nombre string, precio float64, cantidad int) (ItemPedido, error) {
	if cantidad <= 0 {
		return ItemPedido{}, utils.ErrCantidadInvalida
	}
	return ItemPedido{
		productoID: productoID,
		nombre:     nombre,
		precio:     precio,
		cantidad:   cantidad,
	}, nil
}

// ProductoID devuelve el codigo del producto de la linea.
func (i ItemPedido) ProductoID() string { return i.productoID }

// Nombre devuelve el nombre del producto de la linea.
func (i ItemPedido) Nombre() string { return i.nombre }

// Precio devuelve el precio unitario congelado en la linea.
func (i ItemPedido) Precio() float64 { return i.precio }

// Cantidad devuelve las unidades pedidas en la linea.
func (i ItemPedido) Cantidad() int { return i.cantidad }

// Subtotal devuelve el importe de la linea (precio x cantidad).
// float64(i.cantidad) convierte el entero a decimal: Go no permite
// multiplicar un float64 por un int directamente.
func (i ItemPedido) Subtotal() float64 {
	return utils.RedondearDinero(i.precio * float64(i.cantidad))
}

// Pedido representa una compra de un cliente.
// El pedido NO calcula su propio total: ese calculo necesita consultar el
// stock real de los productos, y eso se hace en la capa de logica de negocio.
type Pedido struct {
	id        string
	clienteID string
	items     []ItemPedido // slice: lista que puede crecer
	estado    string
	subtotal  float64
	descuento float64 // porcentaje aplicado
	total     float64
}

// NuevoPedido crea un pedido vacio en estado PENDIENTE.
func NuevoPedido(id, clienteID string) *Pedido {
	return &Pedido{
		id:        id,
		clienteID: clienteID,
		items:     []ItemPedido{}, // empieza como lista vacia
		estado:    EstadoPendiente,
	}
}

// ----- GETTERS -----

// ID devuelve el codigo del pedido.
func (p *Pedido) ID() string { return p.id }

// ClienteID devuelve el codigo del cliente dueno del pedido.
func (p *Pedido) ClienteID() string { return p.clienteID }

// Items devuelve las lineas del detalle.
func (p *Pedido) Items() []ItemPedido { return p.items }

// Estado devuelve el estado actual del pedido.
func (p *Pedido) Estado() string { return p.estado }

// Subtotal devuelve el importe antes del descuento.
func (p *Pedido) Subtotal() float64 { return p.subtotal }

// Descuento devuelve el porcentaje de descuento aplicado.
func (p *Pedido) Descuento() float64 { return p.descuento }

// Total devuelve el importe final a pagar.
func (p *Pedido) Total() float64 { return p.total }

// EsEditable indica si el pedido todavia admite cambios.
func (p *Pedido) EsEditable() bool { return p.estado == EstadoPendiente }

// ----- COMPORTAMIENTO -----

// AgregarItem agrega una linea al pedido. Si el producto ya estaba, suma la
// cantidad a la linea existente en vez de crear una linea repetida.
func (p *Pedido) AgregarItem(item ItemPedido) error {
	if !p.EsEditable() {
		return utils.ErrPedidoCerrado
	}
	// range recorre el slice. Se usa el indice (i) y no una copia del
	// elemento, porque hay que modificar la linea original.
	for i := range p.items {
		if p.items[i].productoID == item.productoID {
			p.items[i].cantidad += item.cantidad
			return nil
		}
	}
	// append agrega al final del slice y devuelve el slice nuevo,
	// por eso hay que reasignarlo a p.items.
	p.items = append(p.items, item)
	return nil
}

// GuardarTotales guarda el resultado del calculo hecho en la logica de negocio.
func (p *Pedido) GuardarTotales(subtotal, descuento, total float64) {
	p.subtotal = subtotal
	p.descuento = descuento
	p.total = total
}

// Confirmar cierra el pedido. Falla si esta vacio o si ya fue confirmado.
func (p *Pedido) Confirmar() error {
	if !p.EsEditable() {
		return utils.ErrPedidoCerrado
	}
	if len(p.items) == 0 {
		return utils.ErrPedidoVacio
	}
	p.estado = EstadoConfirmado
	return nil
}

// Copia devuelve un pedido nuevo con los mismos valores que este.
//
// Tambien copia el slice de lineas con make y copy. Copiar solo el struct no
// bastaria: los slices comparten el arreglo interno, asi que las dos copias
// apuntarian al mismo detalle y modificar una afectaria a la otra.
func (p *Pedido) Copia() *Pedido {
	lineas := make([]ItemPedido, len(p.items))
	copy(lineas, p.items)
	return &Pedido{
		id:        p.id,
		clienteID: p.clienteID,
		items:     lineas,
		estado:    p.estado,
		subtotal:  p.subtotal,
		descuento: p.descuento,
		total:     p.total,
	}
}

// Describir implementa la interfaz Entity para Pedido.
func (p *Pedido) Describir() string {
	return fmt.Sprintf("[PEDIDO]   %-5s cliente:%-5s lineas:%2d  total:$%8.2f  (%s)",
		p.id, p.clienteID, len(p.items), p.total, p.estado)
}

// ==========================================================================
// SERIALIZACION JSON
// ==========================================================================

// itemJSON es el struct puente para serializar una linea del detalle.
type itemJSON struct {
	ProductoID string  `json:"producto_id"`
	Nombre     string  `json:"producto_nombre"`
	Precio     float64 `json:"precio_unitario"`
	Cantidad   int     `json:"cantidad"`
	Subtotal   float64 `json:"subtotal"`
}

// MarshalJSON convierte la linea del detalle a JSON.
// Incluye el subtotal aunque no sea un campo del struct, porque es un dato util
// para quien consume el servicio web y evita que tenga que calcularlo.
func (i ItemPedido) MarshalJSON() ([]byte, error) {
	return json.Marshal(itemJSON{
		ProductoID: i.productoID,
		Nombre:     i.nombre,
		Precio:     i.precio,
		Cantidad:   i.cantidad,
		Subtotal:   i.Subtotal(),
	})
}

// UnmarshalJSON reconstruye una linea del detalle desde JSON.
func (i *ItemPedido) UnmarshalJSON(data []byte) error {
	var aux itemJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("%w: detalle de pedido invalido", utils.ErrArchivoInvalido)
	}
	if aux.Cantidad <= 0 {
		return utils.ErrCantidadInvalida
	}
	i.productoID = aux.ProductoID
	i.nombre = aux.Nombre
	i.precio = aux.Precio
	i.cantidad = aux.Cantidad
	return nil
}

// pedidoJSON es el struct puente para serializar el pedido completo.
type pedidoJSON struct {
	ID        string       `json:"id"`
	ClienteID string       `json:"cliente_id"`
	Items     []ItemPedido `json:"detalle"`
	Estado    string       `json:"estado"`
	Subtotal  float64      `json:"subtotal"`
	Descuento float64      `json:"descuento_porcentaje"`
	Total     float64      `json:"total"`
}

// MarshalJSON convierte el pedido y su detalle a JSON.
//
// El campo Items es un slice de ItemPedido, que a su vez tiene su propio
// MarshalJSON. Cuando encoding/json llega a ese slice, llama al metodo de cada
// elemento, asi que la serializacion se resuelve en cascada automaticamente.
func (p Pedido) MarshalJSON() ([]byte, error) {
	return json.Marshal(pedidoJSON{
		ID:        p.id,
		ClienteID: p.clienteID,
		Items:     p.items,
		Estado:    p.estado,
		Subtotal:  p.subtotal,
		Descuento: p.descuento,
		Total:     p.total,
	})
}

// UnmarshalJSON reconstruye el pedido desde JSON.
func (p *Pedido) UnmarshalJSON(data []byte) error {
	var aux pedidoJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("%w: pedido con formato invalido", utils.ErrArchivoInvalido)
	}
	// Si el archivo traia el detalle vacio, se deja una lista vacia en vez de
	// una lista nula, para que agregar items despues funcione sin problemas.
	if aux.Items == nil {
		aux.Items = []ItemPedido{}
	}
	p.id = aux.ID
	p.clienteID = aux.ClienteID
	p.items = aux.Items
	p.estado = aux.Estado
	p.subtotal = aux.Subtotal
	p.descuento = aux.Descuento
	p.total = aux.Total
	return nil
}
