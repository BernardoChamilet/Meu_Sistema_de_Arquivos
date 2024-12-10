package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	// Verificando se o arquivo meufs.js já existe no diretório atual
	_, erro := os.Stat("meufs.fs")
	if erro != nil {
		// Arquivo não existe
		if os.IsNotExist(erro) {
			// Criando o sistema de arquivos
			if erro = CriarFS(); erro != nil {
				log.Fatal(erro)
			}
		} else {
			log.Fatalf("erro ao verificar existência do arquivo: %v", erro)
		}
	}
	// Abrindo o arquivo para leitura e escrita
	meuFS, erro := os.OpenFile("meufs.fs", os.O_RDWR, 0644)
	if erro != nil {
		log.Fatal(erro)
	}
	defer meuFS.Close()
	// Lendo o cabeçalho
	cabecalho, erro := LerCabecalho(meuFS)
	if erro != nil {
		log.Fatal(erro)
	}
	// Esperando ação do usuário
	var escolha int
	fmt.Println("O que deseja fazer?\n1. Uploadear arquivo\n2. Baixar arquivo\n3. Renomear arquivo\n4. Remover arquivo\n5. Listar arquivos\n6. Mostrar espaço livre:  ")
	_, erro = fmt.Scanf("%d", &escolha)
	if erro != nil {
		log.Fatal(erro)
	}
	// Limpando o buffer de entrada
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n') // Consome o '\n'
	// Opção 1: Copiar arquivo para dentro do meufs.fs
	if escolha == 1 {
		// Solicitando arquivo e o copiando para meufs
		if erro = CopiarParaMeuFS(meuFS, cabecalho); erro != nil {
			log.Fatal(erro)
		}
	} else if escolha == 2 {
		// Solicitando nome do arquivo e o copiando para sistema de arquivos real
		if erro = CopiarParaSistemaReal(meuFS, cabecalho); erro != nil {
			log.Fatal(erro)
		}
	}
}
