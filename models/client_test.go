package models

import (
	"errors"
	"testing"

	"ecommerce/utils"
)

// TestValidarEmail comprueba la validacion de correos electronicos.
//
// La validacion se hizo con el paquete strings en lugar de una expresion
// regular, por lo que conviene probar tanto los correos que deben aceptarse
// como los que deben rechazarse.
func TestValidarEmail(t *testing.T) {
	validos := []string{
		"maria@correo.com",
		"juan.perez@uide.edu.ec",
		"a@b.co",
	}
	for _, email := range validos {
		t.Run("valido: "+email, func(t *testing.T) {
			if err := ValidarEmail(email); err != nil {
				t.Errorf("'%s' deberia ser valido, se obtuvo: %v", email, err)
			}
		})
	}

	invalidos := []struct {
		email  string
		motivo string
	}{
		{"sinarroba.com", "no tiene arroba"},
		{"doble@@correo.com", "tiene dos arrobas"},
		{"@correo.com", "no tiene usuario antes de la arroba"},
		{"maria@correo", "el dominio no tiene punto"},
		{"maria @correo.com", "contiene un espacio"},
		{"", "esta vacio"},
	}
	for _, caso := range invalidos {
		t.Run("invalido: "+caso.motivo, func(t *testing.T) {
			err := ValidarEmail(caso.email)
			if err == nil {
				t.Fatalf("'%s' deberia rechazarse porque %s", caso.email, caso.motivo)
			}
			if !errors.Is(err, utils.ErrEmailInvalido) {
				t.Errorf("se esperaba ErrEmailInvalido, se obtuvo: %v", err)
			}
		})
	}
}

// TestNuevoCliente comprueba la creacion de clientes y la normalizacion del
// correo a minusculas.
func TestNuevoCliente(t *testing.T) {
	cliente, err := NuevoCliente("C001", "Maria Lopez", "MARIA@Correo.COM", "0991234567")
	if err != nil {
		t.Fatalf("no deberia haber error con datos validos: %v", err)
	}
	// El correo se guarda en minusculas para que dos escrituras distintas del
	// mismo buzon no se traten como clientes diferentes.
	if cliente.Email() != "maria@correo.com" {
		t.Errorf("el correo deberia guardarse en minusculas, se obtuvo '%s'", cliente.Email())
	}
}

// TestNuevoClienteInvalido comprueba que se rechacen los datos incorrectos.
func TestNuevoClienteInvalido(t *testing.T) {
	if _, err := NuevoCliente("C001", "ab", "maria@correo.com", "0991234567"); err == nil {
		t.Error("deberia rechazarse un nombre de menos de 3 caracteres")
	}
	if _, err := NuevoCliente("C001", "Maria Lopez", "correomalo", "0991234567"); err == nil {
		t.Error("deberia rechazarse un correo sin formato valido")
	}
}
