// Package storage se encarga de guardar y leer los datos en archivos JSON.
//
// Cada entidad tiene su propio archivo dentro de la carpeta data: products.json,
// clients.json y orders.json. Al iniciar el programa se cargan los datos de esos
// archivos, y cada vez que algo cambia se vuelven a escribir.
//
// Este paquete no sabe nada de las reglas del negocio: solo convierte objetos a
// JSON y JSON a objetos.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ecommerce/models"
	"ecommerce/utils"
)

// Rutas de los archivos donde se guardan los datos.
const (
	ArchivoProductos = "data/products.json"
	ArchivoClientes  = "data/clients.json"
	ArchivoPedidos   = "data/orders.json"
)

// asegurarArchivo crea la carpeta y el archivo si todavia no existen.
//
// El archivo nuevo se crea con el contenido "[]", que es una lista JSON vacia.
// Es necesario porque un archivo de cero bytes haria fallar a json.Unmarshal,
// mientras que "[]" se interpreta correctamente como una lista sin elementos.
func asegurarArchivo(ruta string) error {
	carpeta := filepath.Dir(ruta)
	if err := os.MkdirAll(carpeta, 0755); err != nil {
		return fmt.Errorf("%w: no se pudo crear la carpeta %s", utils.ErrArchivo, carpeta)
	}
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		if err := os.WriteFile(ruta, []byte("[]"), 0644); err != nil {
			return fmt.Errorf("%w: no se pudo crear %s", utils.ErrArchivo, ruta)
		}
	}
	return nil
}

// leerArchivo devuelve el contenido completo del archivo indicado.
func leerArchivo(ruta string) ([]byte, error) {
	if err := asegurarArchivo(ruta); err != nil {
		return nil, err
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: no se pudo leer %s", utils.ErrArchivo, ruta)
	}
	if len(contenido) == 0 {
		return []byte("[]"), nil
	}
	return contenido, nil
}

// escribirArchivo guarda el contenido en el archivo indicado.
func escribirArchivo(ruta string, contenido []byte) error {
	if err := asegurarArchivo(ruta); err != nil {
		return err
	}
	if err := os.WriteFile(ruta, contenido, 0644); err != nil {
		return fmt.Errorf("%w: no se pudo escribir %s", utils.ErrArchivo, ruta)
	}
	return nil
}

// ==========================================================================
// PRODUCTOS
// ==========================================================================

// GuardarProductos escribe la lista de productos en data/products.json.
//
// MarshalIndent genera el JSON con saltos de linea e indentacion de dos
// espacios, para que el archivo sea legible al abrirlo. Cada producto se
// convierte usando el metodo MarshalJSON definido en el modelo, que respeta la
// encapsulacion.
func GuardarProductos(productos []*models.Producto) error {
	contenido, err := json.MarshalIndent(productos, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: no se pudo convertir los productos a JSON", utils.ErrArchivo)
	}
	return escribirArchivo(ArchivoProductos, contenido)
}

// CargarProductos lee los productos guardados en data/products.json.
func CargarProductos() ([]*models.Producto, error) {
	contenido, err := leerArchivo(ArchivoProductos)
	if err != nil {
		return nil, err
	}
	var productos []*models.Producto
	if err := json.Unmarshal(contenido, &productos); err != nil {
		return nil, fmt.Errorf("%w: %s", utils.ErrArchivoInvalido, ArchivoProductos)
	}
	if productos == nil {
		productos = []*models.Producto{}
	}
	return productos, nil
}

// ==========================================================================
// CLIENTES
// ==========================================================================

// GuardarClientes escribe la lista de clientes en data/clients.json.
func GuardarClientes(clientes []*models.Cliente) error {
	contenido, err := json.MarshalIndent(clientes, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: no se pudo convertir los clientes a JSON", utils.ErrArchivo)
	}
	return escribirArchivo(ArchivoClientes, contenido)
}

// CargarClientes lee los clientes guardados en data/clients.json.
func CargarClientes() ([]*models.Cliente, error) {
	contenido, err := leerArchivo(ArchivoClientes)
	if err != nil {
		return nil, err
	}
	var clientes []*models.Cliente
	if err := json.Unmarshal(contenido, &clientes); err != nil {
		return nil, fmt.Errorf("%w: %s", utils.ErrArchivoInvalido, ArchivoClientes)
	}
	if clientes == nil {
		clientes = []*models.Cliente{}
	}
	return clientes, nil
}

// ==========================================================================
// PEDIDOS
// ==========================================================================

// GuardarPedidos escribe la lista de pedidos en data/orders.json.
func GuardarPedidos(pedidos []*models.Pedido) error {
	contenido, err := json.MarshalIndent(pedidos, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: no se pudo convertir los pedidos a JSON", utils.ErrArchivo)
	}
	return escribirArchivo(ArchivoPedidos, contenido)
}

// CargarPedidos lee los pedidos guardados en data/orders.json.
func CargarPedidos() ([]*models.Pedido, error) {
	contenido, err := leerArchivo(ArchivoPedidos)
	if err != nil {
		return nil, err
	}
	var pedidos []*models.Pedido
	if err := json.Unmarshal(contenido, &pedidos); err != nil {
		return nil, fmt.Errorf("%w: %s", utils.ErrArchivoInvalido, ArchivoPedidos)
	}
	if pedidos == nil {
		pedidos = []*models.Pedido{}
	}
	return pedidos, nil
}
