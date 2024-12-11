package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
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
	Protegido   uint8
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
	// Limpando o buffer de entrada
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n') // Consome o '\n'
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
	novaEntradaRoot.Protegido = 0
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

// CopiarParaSistemaReal copia um arquivo de dentro do meuFS para um sistema de arquivos real (disco, pendrive, etc)
func CopiarParaSistemaReal(meuFS *os.File, cabecalho Cabecalho) error {
	// Solicitando nome do arquivo a ser copiado para o sistema real
	var nomeArquivo string
	fmt.Println("Digite o nome do arquivo que deseja baixar: ")
	fmt.Scanln(&nomeArquivo)
	// Lendo root
	root, erro := LerRoot(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Vendo se arquivo existe no root
	var indiceDoArquivoNoRoot int = -1
	for indice, entrada := range root {
		if strings.TrimRight(string(entrada.NomeArquivo[:]), "\x00") == nomeArquivo {
			indiceDoArquivoNoRoot = indice
			break
		}
	}
	if indiceDoArquivoNoRoot == -1 {
		return errors.New("arquivo com esse nome não existe no sistema de arquivos meufs")
	}
	// Lendo FAT
	fat, erro := LerFAT(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Obtendo endereço dos blocos do arquivo com a fat
	posicaoNaFAT := root[indiceDoArquivoNoRoot].EnderecoFAT
	var blocosDoArquivo []uint32
	blocosDoArquivo = append(blocosDoArquivo, posicaoNaFAT)
	for {
		if fat[posicaoNaFAT] == 0xFFFFFFFF {
			break
		}
		posicaoNaFAT = fat[posicaoNaFAT]
		blocosDoArquivo = append(blocosDoArquivo, posicaoNaFAT)
	}
	// Solicitando onde no sistema real o arquivo vai ser copiado para
	var caminho string
	fmt.Println("Digite onde você deseja que o arquivo seja baixado: ")
	fmt.Scanln(&caminho)
	var nomeReal string
	fmt.Println("Digite que nome deseja dar ao arquivo baixado: ")
	fmt.Scanln(&nomeReal)
	// Criar o arquivo no sistema real
	caminhoComleto := fmt.Sprintf("%s/%s", caminho, nomeReal)
	arquivoReal, erro := os.Create(caminhoComleto)
	if erro != nil {
		return fmt.Errorf("erro ao criar o arquivo no sistema real: %w", erro)
	}
	defer arquivoReal.Close()
	// Passando para o sistema real os blocos 1 por 1 da área de dados do meufs
	blocoDoArquivo := make([]byte, cabecalho.TamanhoBloco)
	for _, entrada := range blocosDoArquivo {
		// Obtendo posicao do bloco
		posicaoBloco := int64(cabecalho.InicioDados + (cabecalho.TamanhoBloco * entrada))
		// Movendo ponteiro para a posicao do bloco do arquivo
		_, erro = meuFS.Seek(posicaoBloco, 0)
		if erro != nil {
			return fmt.Errorf("erro ao posicionar ponteiro no bloco do arquivo: %w", erro)
		}
		// Lendo bloco
		numBytes, erro := meuFS.Read(blocoDoArquivo)
		if erro != nil {
			return fmt.Errorf("erro ao ler bloco do arquivo a ser baixado: %w", erro)
		}
		// Escrevendo no sistemas de arquivo real
		// Necessita de :numBytes para escrever apenas os bytes usados pelo arquivo no último bloco
		_, erro = arquivoReal.Write(blocoDoArquivo[:numBytes])
		if erro != nil {
			return fmt.Errorf("erro ao escrever bloco no arquivo no sistema real: %w", erro)
		}
	}
	fmt.Println("arquivo baixado com sucesso!")
	return nil
}

// RenomearArquivo renomeia um arquivo armazenado dentro do meufs
func RenomearArquivo(meuFS *os.File, cabecalho Cabecalho) error {
	// Solicitando nome do arquivo a ser renomeado
	var nomeAntigo string
	fmt.Println("Digite o nome do arquivo que deseja renomear: ")
	fmt.Scanln(&nomeAntigo)
	// Solicitando novo nome do arquivo a ser renomeado
	var nomeNovo string
	fmt.Println("Digite o novo nome do arquivo que deseja renomear: ")
	fmt.Scanln(&nomeNovo)
	if len(nomeNovo) > 19 {
		return errors.New("nome do arquivo nao pode ter mais de 19 caracteres")
	}
	// Lendo root
	root, erro := LerRoot(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Vendo se arquivo existe no root
	var indiceDoArquivoNoRoot int = -1
	for indice, entrada := range root {
		if strings.TrimRight(string(entrada.NomeArquivo[:]), "\x00") == nomeAntigo {
			indiceDoArquivoNoRoot = indice
			break
		}
	}
	if indiceDoArquivoNoRoot == -1 {
		return errors.New("arquivo com esse nome não existe no sistema de arquivos meufs")
	}
	// Renomeando arquivo
	var nomeNovoArray [20]byte
	copy(nomeNovoArray[:], nomeNovo)
	root[indiceDoArquivoNoRoot].NomeArquivo = nomeNovoArray
	// Salvando root atualizado
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
	// Garante que os dados estejam no disco
	erro = meuFS.Sync()
	if erro != nil {
		return fmt.Errorf("erro ao sincronizar o arquivo: %w", erro)
	}
	fmt.Println("arquivo renomeado com sucesso!")
	return nil
}

// RemoverArquivo remove um arquivo de dentro do meufs
func RemoverArquivo(meuFS *os.File, cabecalho Cabecalho) error {
	// Solicitando nome do arquivo a ser renomeado
	var nomeArquivo string
	fmt.Println("Digite o nome do arquivo que deseja remover: ")
	fmt.Scanln(&nomeArquivo)
	// Lendo root
	root, erro := LerRoot(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Vendo se arquivo existe no root
	var indiceDoArquivoNoRoot int = -1
	for indice, entrada := range root {
		if strings.TrimRight(string(entrada.NomeArquivo[:]), "\x00") == nomeArquivo {
			indiceDoArquivoNoRoot = indice
			break
		}
	}
	if indiceDoArquivoNoRoot == -1 {
		return errors.New("arquivo com esse nome não existe no sistema de arquivos meufs")
	}
	// Vendo se arquivo é protegido
	if root[indiceDoArquivoNoRoot].Protegido == 1 {
		return errors.New("esse arquivo está protegido de ser excluido")
	}
	// Lendo FAT
	fat, erro := LerFAT(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	// Obtendo endereço dos blocos do arquivo com a fat
	posicaoNaFAT := root[indiceDoArquivoNoRoot].EnderecoFAT
	var blocosDoArquivo []uint32
	blocosDoArquivo = append(blocosDoArquivo, posicaoNaFAT)
	for {
		if fat[posicaoNaFAT] == 0xFFFFFFFF {
			break
		}
		posicaoNaFAT = fat[posicaoNaFAT]
		blocosDoArquivo = append(blocosDoArquivo, posicaoNaFAT)
	}
	// Colocando zeros no lugar dos blocos do arquivo e atualizando fat
	blocoDeZeros := make([]byte, cabecalho.TamanhoBloco)
	for _, entrada := range blocosDoArquivo {
		// Obtendo posicao do bloco
		posicaoBloco := int64(cabecalho.InicioDados + (cabecalho.TamanhoBloco * entrada))
		// Movendo ponteiro para a posicao do bloco do arquivo
		_, erro = meuFS.Seek(posicaoBloco, 0)
		if erro != nil {
			return fmt.Errorf("erro ao posicionar ponteiro no bloco do arquivo: %w", erro)
		}
		// Escrevendo zeros no bloco
		_, erro = meuFS.Write(blocoDeZeros)
		if erro != nil {
			return fmt.Errorf("erro ao sobrescrever bloco do arquivo com zeros: %w", erro)
		}
		// Atualizando FAT
		fat[entrada] = 0
	}
	// Atualizando root e o salvando no arquivo
	root[indiceDoArquivoNoRoot] = DiretorioRoot{}
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
	// Salvando FAT atualizada
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
	fmt.Println("arquivo removido com sucesso!")
	return nil
}

// ListarArquivos imprime todos os arquivos armazenados no meufs
func ListarArquivos(meuFS *os.File, cabecalho Cabecalho) error {
	root, erro := LerRoot(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	nenhumArquivo := true
	for _, entrada := range root {
		if entrada.NomeArquivo[0] != 0 {
			nenhumArquivo = false
			fmt.Printf("%s\n", strings.TrimRight(string(entrada.NomeArquivo[:]), "\x00"))
		}
	}
	if nenhumArquivo {
		return errors.New("nenhum arquivo armazenado no sistema de arquivos meufs")
	}
	return nil
}

// MostrarEspacoLivre mostra quantos MB livres tem em relação ao total
func MostrarEspacoLivre(meuFS *os.File, cabecalho Cabecalho) error {
	// Calculando espaço de dados
	espacoDados := cabecalho.TamanhoMeuFS - cabecalho.InicioDados
	// Lendo fat para ver espaços livres
	fat, erro := LerFAT(cabecalho, meuFS)
	if erro != nil {
		return erro
	}
	var espacoLivre uint32 = 0
	for _, entrada := range fat {
		if entrada == 0 {
			espacoLivre += cabecalho.TamanhoBloco
		}
	}
	espacoDados = espacoDados / (1024 * 1024)
	espacoLivre = espacoLivre / (1024 * 1024)
	fmt.Printf("%dMB livres de %dMB\n", espacoLivre, espacoDados)
	return nil
}
