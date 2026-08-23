package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"ecommerce/utils"
)

// Cliente representa a una persona registrada en el sistema.
type Cliente struct {
	id       string
	nombre   string
	email    string
	telefono string
}

// NuevoCliente es el constructor del cliente y valida sus datos.
func NuevoCliente(id, nombre, email, telefono string) (*Cliente, error) {
	nombre = strings.TrimSpace(nombre)
	if len(nombre) < 3 {
		return nil, utils.ErrNombreInvalido
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if err := ValidarEmail(email); err != nil {
		return nil, err
	}
	return &Cliente{
		id:       id,
		nombre:   nombre,
		email:    email,
		telefono: strings.TrimSpace(telefono),
	}, nil
}

// ValidarEmail revisa el formato del correo usando el paquete strings.
// No se usa una expresion regular porque ese tema no se ha visto en clase.
func ValidarEmail(email string) error {
	// Debe tener exactamente una arroba y ningun espacio.
	if strings.Count(email, "@") != 1 || strings.Contains(email, " ") {
		return utils.ErrEmailInvalido
	}
	// Split corta el texto en la arroba: partes[0] es el usuario, partes[1] el dominio.
	partes := strings.Split(email, "@")
	usuario := partes[0]
	dominio := partes[1]
	if usuario == "" || !strings.Contains(dominio, ".") {
		return utils.ErrEmailInvalido
	}
	return nil
}

// ----- GETTERS -----

// ID devuelve el codigo del cliente.
func (c *Cliente) ID() string { return c.id }

// Nombre devuelve el nombre del cliente.
func (c *Cliente) Nombre() string { return c.nombre }

// Email devuelve el correo del cliente.
func (c *Cliente) Email() string { return c.email }

// Telefono devuelve el telefono del cliente.
func (c *Cliente) Telefono() string { return c.telefono }

// ----- SETTERS -----

// CambiarNombre modifica el nombre previa validacion.
func (c *Cliente) CambiarNombre(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if len(nombre) < 3 {
		return utils.ErrNombreInvalido
	}
	c.nombre = nombre
	return nil
}

// CambiarEmail modifica el correo previa validacion de formato.
func (c *Cliente) CambiarEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := ValidarEmail(email); err != nil {
		return err
	}
	c.email = email
	return nil
}

// CambiarTelefono modifica el telefono del cliente.
func (c *Cliente) CambiarTelefono(telefono string) error {
	c.telefono = strings.TrimSpace(telefono)
	return nil
}

// Copia devuelve un cliente nuevo con los mismos valores que este.
// Cumple el mismo proposito que el metodo Copia de Producto: entregar datos
// seguros de leer aunque otra goroutine este modificando el original.
func (c *Cliente) Copia() *Cliente {
	return &Cliente{
		id:       c.id,
		nombre:   c.nombre,
		email:    c.email,
		telefono: c.telefono,
	}
}

// Describir implementa la interfaz Entity para Cliente.
func (c *Cliente) Describir() string {
	return fmt.Sprintf("[CLIENTE]  %-5s %-22s %-25s tel:%s",
		c.id, c.nombre, c.email, c.telefono)
}

// ==========================================================================
// SERIALIZACION JSON
// ==========================================================================

// clienteJSON es el struct puente entre el cliente encapsulado y el JSON.
// Cumple el mismo proposito que productoJSON: permitir la serializacion sin
// hacer publicos los campos del struct Cliente.
type clienteJSON struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"correo"`
	Telefono string `json:"telefono"`
}

// MarshalJSON convierte el cliente a JSON sin exponer sus campos privados.
func (c Cliente) MarshalJSON() ([]byte, error) {
	return json.Marshal(clienteJSON{
		ID:       c.id,
		Nombre:   c.nombre,
		Email:    c.email,
		Telefono: c.telefono,
	})
}

// UnmarshalJSON reconstruye un cliente desde JSON validando los datos leidos.
func (c *Cliente) UnmarshalJSON(data []byte) error {
	var aux clienteJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("%w: cliente con formato invalido", utils.ErrArchivoInvalido)
	}
	if len(strings.TrimSpace(aux.Nombre)) < 3 {
		return utils.ErrNombreInvalido
	}
	if err := ValidarEmail(aux.Email); err != nil {
		return err
	}
	c.id = aux.ID
	c.nombre = aux.Nombre
	c.email = aux.Email
	c.telefono = aux.Telefono
	return nil
}
