# Sistema de Gerenciamento de Conteúdo com Hash SHA-256

Este projeto implementa um sistema simples mas poderoso de gerenciamento de conteúdo utilizando hashes SHA-256 para organização e acesso de arquivos.

## Funcionalidades

### 1. Sistema de Upload e Categorização
- Upload de arquivos e textos
- Categorização automática por hash SHA-256
- Relações entre conteúdos e categorias
- Interface de navegação entre categorias e conteúdos relacionados

### 2. Sistema de Download e Indexação de URLs
- Download automático de conteúdo a partir de URLs
- Categorização por tipo de arquivo e data
- Armazenamento de metadados em JSON
- Suporte para informações adicionais como título, descrição e autor

## Como Funciona

### Sistema de Upload
O sistema utiliza hash SHA-256 para identificar de forma única tanto arquivos quanto categorias:

1. Ao fazer upload de um conteúdo, o sistema:
   - Gera um hash SHA-256 do conteúdo
   - Cria uma pasta com o nome do hash
   - Armazena o conteúdo dentro dessa pasta
   - Vincula esse conteúdo à categoria especificada (também identificada por hash)

2. A categorização permite:
   - Navegar entre conteúdos relacionados
   - Encontrar conteúdo através de sua categoria
   - Manter relações bidirecionais entre categorias e conteúdos

### Sistema de Download de URLs
O sistema permite salvar conteúdo da web de forma organizada:

1. Ao enviar uma URL, o sistema:
   - Gera um hash SHA-256 da URL
   - Baixa o conteúdo da página
   - Categoriza por tipo de arquivo e data
   - Salva metadados adicionais em formato JSON

2. A organização é feita por:
   - Tipo de arquivo (extensão)
   - Data de download
   - Categoria personalizada (se fornecida)

## Requisitos

- PHP 7.0 ou superior
- Extensão cURL habilitada
- Permissões de escrita nos diretórios do projeto

## Segurança

- O sistema bloqueia o upload de arquivos PHP por motivos de segurança
- Utiliza sanitização de entrada para evitar injeção de código
- Implementa verificações de validação para garantir a integridade dos dados

## Contribuições

Contribuições são bem-vindas! Sinta-se à vontade para abrir issues ou enviar pull requests para melhorar este projeto.

## 📝 Licença

MIT

---

Desenvolvido com ❤️ por Arthur S. Sacramento
