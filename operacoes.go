package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

type Cabecalho struct {
	TamanhoCabecalho uint32
	TamanhoBloco     uint32
	TamanhoMeuFS     uint32
	InicioFAT        uint32
	InicioRoot       uint32
	InicioDados      uint32
}

type DiretorioRoot struct {
	NomeArquivo [20]byte // Máximo 19 caracteres
	EnderecoFAT uint32
}

// CriarFS cria um arquivo meufs.fs com tamanho especificado pelo usuário e escreve o cabeçalho do sistema de arquivos nele
func CriarFS() error {
	// Pedindo ao usuário para informar tamanho total do sistema de arquivos
	var tamanhoArquivoMB int
	fmt.Println("Escolha o tamanho do sistema de arquivos em MB(mínimo: 100MB, máximo: 800MB): ")
	_, erro := fmt.Scanf("%d", &tamanhoArquivoMB)
	if erro != nil {
		return fmt.Errorf("erro ao ler a entrada: %w", erro)
	}
	// Validando o tamanho fornecido pelo usuário
	if tamanhoArquivoMB < 100 || tamanhoArquivoMB > 800 {
		return errors.New("o tamanho deve estar entre 100MB e 800MB")
	}
	// Criando o sistema de arquivos meufs.fs
	meuFS, erro := os.Create("meufs.fs")
	if erro != nil {
		return fmt.Errorf("falha ao criar o arquivo: %w", erro)
	}
	defer meuFS.Close()
	// Definindo tamanho do arquivo
	tamanhoArquivoBytes := int64(tamanhoArquivoMB) * 1024 * 1024
	if erro = meuFS.Truncate(tamanhoArquivoBytes); erro != nil {
		return fmt.Errorf("erro ao definir o tamanho do arquivo: %v", erro)
	}
	fmt.Printf("Sistema de arquivos criado com tamanho de %dMB\n", tamanhoArquivoMB)
	// Criando cabeçalho do sistema de arquivos e o escrevendo no meuFS
	// Estrutura do meufs: cabeçalho root tad dados nessa ordem
	// Calculando os endereços segundo estrutura acima
	tamanhoBloco := uint32(4 * 1024) // 4kb
	tamanhoCabecalho := uint32(binary.Size(Cabecalho{}))
	inicioRoot := tamanhoCabecalho
	tamanhoRoot := uint32(binary.Size(DiretorioRoot{})) * 200 // máximo 200 arquivos
	inicioFAT := inicioRoot + tamanhoRoot
	// 4/4100 avos do espaço disponível após inserir cabeçalho e root. (4 bytes de fat para cada 4096 bytes (1 bloco) de dados)
	tamanhoFAT := ((uint32(tamanhoArquivoBytes) - tamanhoCabecalho - tamanhoRoot) * 4) / (tamanhoBloco + 4)
	inicioDados := inicioFAT + tamanhoFAT
	// Criando cabeçalho
	cabecalho := Cabecalho{
		TamanhoCabecalho: tamanhoCabecalho,
		TamanhoMeuFS:     uint32(tamanhoArquivoBytes),
		TamanhoBloco:     tamanhoBloco,
		InicioRoot:       inicioRoot,
		InicioFAT:        inicioFAT,
		InicioDados:      inicioDados,
	}
	// Movendo ponteiro para o início do meufs.fs
	_, erro = meuFS.Seek(0, 0)
	if erro != nil {
		return fmt.Errorf("erro ao achar início do arquivo: %w", erro)
	}
	// Escrevendo cabeçalho no formato binário
	erro = binary.Write(meuFS, binary.LittleEndian, &cabecalho)
	if erro != nil {
		return fmt.Errorf("erro ao escrever cabecalho: %w", erro)
	}
	// Garante que os dados estejam no disco
	erro = meuFS.Sync()
	if erro != nil {
		return fmt.Errorf("erro ao sincronizar o arquivo: %w", erro)
	}
	fmt.Println("Cabeçalho criado e escrito no sistemas de arquivos com sucesso")
	return nil
}

// LerCabecalho lê o cabeçalho e o mapeia para um struct
func LerCabecalho(arquivo *os.File) (Cabecalho, error) {
	// Movendo ponteiro para o inicio do arquivo
	var cabecalho Cabecalho
	_, erro := arquivo.Seek(0, 0)
	if erro != nil {
		return Cabecalho{}, fmt.Errorf("erro ao posicionar o ponteiro: %v", erro)
	}
	// Lendo cabeçalho e o mapeando para struct
	erro = binary.Read(arquivo, binary.LittleEndian, &cabecalho)
	if erro != nil {
		return Cabecalho{}, fmt.Errorf("erro ao ler o cabeçalho: %v", erro)
	}
	return cabecalho, nil
}

// LerFAT lê FAT a mapeando para um slice
func LerFAT(cabecalho Cabecalho, meuFS *os.File) ([]uint32, error) {
	// Vendo quantas entradas a fat tem
	numEntradasFAT := (cabecalho.TamanhoMeuFS - cabecalho.InicioDados) / cabecalho.TamanhoBloco
	// Criando slice FAT
	fat := make([]uint32, numEntradasFAT)
	// Posicionando ponteiro no início da FAT
	_, erro := meuFS.Seek(int64(cabecalho.InicioFAT), 0)
	if erro != nil {
		return nil, fmt.Errorf("erro ao posicionar o ponteiro no início da FAT: %w", erro)
	}
	// Lendo FAT
	erro = binary.Read(meuFS, binary.LittleEndian, &fat)
	if erro != nil {
		return nil, fmt.Errorf("erro ao ler a FAT: %w", erro)
	}
	return fat, nil
}

// LerRoot lê o diretório raiz o mapeando para um slice
func LerRoot(cabecalho Cabecalho, meuFS *os.File) ([]DiretorioRoot, error) {
	// Criando sliec root
	root := make([]DiretorioRoot, 200) // 200 entradas
	// Posicionando ponteiro no inicio do root
	_, erro := meuFS.Seek(int64(cabecalho.InicioRoot), 0)
	if erro != nil {
		return nil, fmt.Errorf("erro ao posicionar o ponteiro no início do diretorio raiz: %w", erro)
	}
	// Lendo root
	erro = binary.Read(meuFS, binary.LittleEndian, &root)
	if erro != nil {
		return nil, fmt.Errorf("erro ao ler diretorio raiz: %w", erro)
	}
	return root, nil
}

// CopiarParaMeuFS copia um arquivo escolhido pelo usuário para o sistema de arquivos meufs
func CopiarParaMeuFS(meuFS *os.File, cabecalho Cabecalho) error {
	// Solicitando caminho e nome do arquivo
	var caminho string
	fmt.Println("Digite o caminho do arquivo que deseja copiar para o meufs.fs: ")
	fmt.Scanln(&caminho)
	var nomeArquivo string
	fmt.Println("Digite o nome que quer dar ao arquivo: ")
	fmt.Scanln(&nomeArquivo)
	if len(nomeArquivo) > 19 {
		return errors.New("nome do arquivo nao pode ter mais de 19 caracteres")
	}
	// Abrindo arquivo novo
	arquivoNovo, erro := os.Open(caminho)
	if erro != nil {
		return fmt.Errorf("erro ao abrir arquivo a ser guardado: %w", erro)
	}
	defer arquivoNovo.Close()
	// Obtendo tamanho do arquivo novo
	info, erro := arquivoNovo.Stat()
	if erro != nil {
		return fmt.Errorf("erro ao obter informações do arquivo a ser guardado: %w", erro)
	}
	tamanhoArquivo := info.Size()
	if tamanhoArquivo == 0 {
		return errors.New("arquivo escolhido não pode estar vazio")
	}
	// Calculando quantos blocos o arquivo ocupará
	// converto o tamanho do bloco para int64 pois o tamanho do arquivo informado pode ser maior que 4gb
	numBlocosArquivo := uint32((tamanhoArquivo + int64(cabecalho.TamanhoBloco) - 1) / int64(cabecalho.TamanhoBloco))
	// Lendo FAT
	fat, erro := LerFAT(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Vendo se tem espaço livre para guardar o arquivo
	cabe := false
	indicesLivresFAT := make([]uint32, 0, numBlocosArquivo)
	for indice, entrada := range fat {
		if entrada == 0 {
			indicesLivresFAT = append(indicesLivresFAT, uint32(indice))
		}
		if uint32(len(indicesLivresFAT)) == numBlocosArquivo {
			cabe = true
			break
		}
	}
	if !cabe {
		return errors.New("arquivo não coube no sistema de arquivos")
	}
	// Lendo root
	root, erro := LerRoot(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Vendo se tem espaço livre em root
	var indiceLivreRoot int = -1
	for indice, entrada := range root {
		if string(entrada.NomeArquivo[:]) == nomeArquivo {
			return fmt.Errorf("um arquivo com o nome '%s' já existe no sistema", nomeArquivo)
		}
		if entrada.NomeArquivo[0] == 0 && indiceLivreRoot == -1 {
			indiceLivreRoot = indice
		}
	}
	if indiceLivreRoot == -1 {
		return errors.New("diretorio raiz cheio, maximo de 200 arquivos atingido")
	}
	// Separando o arquivo em blocos e os colocando no meufs.fs
	blocoDoArquivo := make([]byte, cabecalho.TamanhoBloco)
	for _, indiceLivre := range indicesLivresFAT {
		// Pegando um bloco do arquivo
		numBytes, erro := arquivoNovo.Read(blocoDoArquivo)
		if erro != nil {
			return fmt.Errorf("erro ao ler arquivo a ser guardado: %w", erro)
		}
		// Movendo ponteiro para posição que bloco será guardado
		posicao := int64(cabecalho.InicioDados + (indiceLivre * cabecalho.TamanhoBloco))
		_, erro = meuFS.Seek(posicao, 0)
		if erro != nil {
			return fmt.Errorf("erro ao posicionar ponteiro no meufs: %w", erro)
		}
		// Escrevendo bloco
		// Necessita de :numBytes para escrever apenas os bytes usados pelo arquivo no último bloco
		_, erro = meuFS.Write(blocoDoArquivo[:numBytes])
		if erro != nil {
			return fmt.Errorf("erro ao escrever bloco no meufs: %w", erro)
		}
	}
	// Atualizando root
	var novaEntradaRoot DiretorioRoot
	copy(novaEntradaRoot.NomeArquivo[:], nomeArquivo)
	novaEntradaRoot.EnderecoFAT = indicesLivresFAT[0]
	root[indiceLivreRoot] = novaEntradaRoot
	// movendo ponteiro
	_, erro = meuFS.Seek(int64(cabecalho.InicioRoot), 0)
	if erro != nil {
		return fmt.Errorf("erro ao posicionar o ponteiro no início do diretorio raiz: %w", erro)
	}
	// escrevendo root atualizado
	erro = binary.Write(meuFS, binary.LittleEndian, root)
	if erro != nil {
		return fmt.Errorf("erro ao escrever root atualizado: %w", erro)
	}
	// Atualizando fat
	for i := 0; i < len(indicesLivresFAT)-1; i++ {
		fat[indicesLivresFAT[i]] = indicesLivresFAT[i+1]
	}
	fat[indicesLivresFAT[len(indicesLivresFAT)-1]] = 0xFFFFFFFF //numero hexadecimal uint32 muito maior que len da fat
	// movendo ponteiro
	_, erro = meuFS.Seek(int64(cabecalho.InicioFAT), 0)
	if erro != nil {
		return fmt.Errorf("erro ao posicionar ponteiro no inicio da FAT: %w", erro)
	}
	// escrevendo fat atualizada
	erro = binary.Write(meuFS, binary.LittleEndian, fat)
	if erro != nil {
		return fmt.Errorf("erro ao escrever FAT atualizada: %w", erro)
	}
	// Garante que os dados estejam no disco
	erro = meuFS.Sync()
	if erro != nil {
		return fmt.Errorf("erro ao sincronizar o arquivo: %w", erro)
	}
	fmt.Println("arquivo copiado com sucesso!")
	return nil
}