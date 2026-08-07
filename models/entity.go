// Package models contiene las entidades del sistema: Producto, Cliente y Pedido.
//
// Todas usan encapsulacion: sus campos empiezan en minuscula (son privados) y
// solo se leen o modifican con getters y setters que validan los datos.
package models

// Entity es la interfaz que cumplen Producto, Cliente y Pedido.
//
// Cualquier tipo que tenga los metodos ID() y Describir() es un Entity, sin
// necesidad de declararlo. Sirve para guardar productos, clientes y pedidos en
// una misma lista y recorrerlos con un solo bucle: eso es polimorfismo.
type Entity interface {
	ID() string
	Describir() string
}
