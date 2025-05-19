# Sistema de Armazenamento e Categorização de Conteúdo

Este sistema PHP permite o armazenamento, categorização e recuperação de conteúdo através de hashes SHA-256, oferecendo uma forma organizada de gerenciar arquivos e textos com metadados associados.

## Visão Geral

O sistema implementa uma solução para armazenar conteúdo (arquivos ou texto) em categorias, utilizando hashes SHA-256 para identificação única. Cada conteúdo é armazenado em uma estrutura de diretórios baseada em seu hash, permitindo fácil recuperação e organização. O sistema também suporta metadados e informações de propriedade (BTC).

## Funcionalidades Principais

- **Upload de Arquivos**: Permite o upload de arquivos com verificação de segurança (bloqueio de arquivos PHP).
- **Entrada de Texto**: Possibilita salvar conteúdo de texto diretamente através de um formulário.
- **Categorização**: Todo conteúdo é associado a uma categoria, facilitando a organização.
- **Busca por Hash**: Implementa um sistema de busca que aceita tanto texto normal quanto hashes SHA-256.
- **Metadados**: Suporta armazenamento de metadados como usuário, título, descrição e URL.
- **Informações de Propriedade**: Permite associar endereços BTC ao conteúdo.
- **Navegação por Links**: Cria automaticamente páginas de índice com links para o conteúdo.

## Estrutura de Diretórios

O sistema utiliza a seguinte estrutura de diretórios:

- `data_tmp/`: Diretório base para todos os uploads
  - `[hash_do_arquivo]/`: Subdiretório para cada arquivo/conteúdo
    - `[hash_do_arquivo].[extensão]`: Arquivo de conteúdo
    - `index.html`: Página de índice com links para o conteúdo
  - `[hash_da_categoria]/`: Subdiretório para cada categoria
    - `[hash_do_arquivo].[extensão]`: Arquivo vazio (referência)
    - `index.html`: Página de índice com links para o conteúdo
- `owners/`: Diretório para informações de BTC
  - `[hash_do_arquivo]`: Arquivo contendo informação BTC
- `metadata/`: Diretório para arquivos de metadados
  - `[hash_do_arquivo].json`: Arquivo JSON com metadados

## Requisitos

- Servidor web com suporte a PHP
- Permissões de escrita nos diretórios do sistema
- Navegador web moderno

## Como Usar

### Upload de Conteúdo

1. Acesse o formulário de upload (index.php)
2. Selecione um arquivo para upload ou insira texto no campo de conteúdo
3. Defina uma categoria para o conteúdo
4. Opcionalmente, preencha os campos de metadados (usuário, título, descrição, URL)
5. Opcionalmente, forneça informações de BTC
6. Envie o formulário

### Busca de Conteúdo

1. Use o formulário de busca
2. Insira texto normal (será convertido para hash SHA-256) ou um hash SHA-256 válido
3. O sistema redirecionará para a página correspondente se existir

### Resposta a Conteúdo

O sistema permite responder a conteúdo existente através do link "Reply", que pré-preenche o formulário com o hash do conteúdo original.

## Segurança

O sistema implementa algumas medidas de segurança:

- Bloqueio de upload de arquivos PHP
- Verificação para evitar que a categoria seja idêntica ao conteúdo
- Uso de hashes SHA-256 para identificação de conteúdo
- Validação de entradas

## Personalização

O sistema utiliza arquivos CSS e JavaScript externos para estilização e funcionalidade:

- `default.css`: Estilos padrão
- `default.js`: Funcionalidades JavaScript
- `ads.js`: Script para gerenciamento de anúncios

## Limitações e Considerações

- O sistema não implementa autenticação de usuários
- Arquivos com conteúdo idêntico (mesmo hash) não podem ser enviados novamente
- O sistema não verifica o tamanho dos arquivos enviados
- Não há validação avançada de tipos de arquivo além do bloqueio de arquivos PHP

## Exemplo de Fluxo de Uso

1. Um usuário envia um arquivo de texto com a categoria "documentos"
2. O sistema calcula o hash SHA-256 do arquivo e da categoria
3. O arquivo é salvo em `data_tmp/[hash_do_arquivo]/[hash_do_arquivo].txt`
4. Um arquivo vazio é criado em `data_tmp/[hash_da_categoria]/[hash_do_arquivo].txt`
5. Páginas de índice são atualizadas com links para o conteúdo
6. Metadados são salvos em `metadata/[hash_do_arquivo].json`
7. Informações BTC são salvas em `owners/[hash_do_arquivo]`
8. O usuário pode acessar o conteúdo através das páginas de índice ou busca

