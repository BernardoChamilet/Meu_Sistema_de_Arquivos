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
	for {
		// Esperando ação do usuário
		var escolha int
		fmt.Println("O que deseja fazer?\n1. Uploadear arquivo\n2. Baixar arquivo\n3. Renomear arquivo\n4. Remover arquivo\n5. Listar arquivos\n6. Mostrar espaço livre\n7. Proteger/Desproteger arquivo (se o arquivo estiver protegido ele será desprotegido e vice-versa)\n8. Criar diretório\n9. Encerrar programa")
		_, erro = fmt.Scanf("%d", &escolha)
		if erro != nil {
			log.Fatal(erro)
		}
		// Limpando o buffer de entrada
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n') // Consome o '\n'
		switch {
		// Opção 1: Copiar arquivo para dentro do meufs.fs
		case escolha == 1:
			if erro = CopiarParaMeuFS(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 2: Copiar arquivo de dentro do meufs.fs para sistema de arquivos real
		case escolha == 2:
			if erro = CopiarParaSistemaReal(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 3: Renomear arquivo armazenado no meufs
		case escolha == 3:
			if erro = RenomearArquivo(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 4: Remover arquivo armazenado no meufs
		case escolha == 4:
			if erro = RemoverArquivo(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 5: Listar todos arquivos armazenados no meufs
		case escolha == 5:
			if erro = ListarArquivos(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 6: Mostrar espaço livre do meufs
		case escolha == 6:
			if erro = MostrarEspacoLivre(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 7: Proteger/Desproteger arquivo
		case escolha == 7:
			if erro = ProtegerDesprotegerArquivo(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 8: Criar diretório
		case escolha == 8:
			if erro = CriarDiretorio(meuFS, cabecalho); erro != nil {
				fmt.Printf("%v\n", erro)
			}
		// Opção 9: Encerrar programa
		case escolha == 9:
			fmt.Println("Programa encerrado")
			return
		// Default: Opção inválida
		default:
			fmt.Println("Opção inválida")
		}
	}
}
