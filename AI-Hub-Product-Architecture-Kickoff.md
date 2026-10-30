# AI Hub --- Product & Technical Kickoff

## Documento de visão, arquitetura e plano inicial de implementação

**Status:** baseline inicial para início do desenvolvimento\
**Versão:** 0.1\
**Data de referência:** 30/10/2026\
**Estratégia de implementação:** Gateway First\
**Primeiro harness:** OpenAI Codex\
**Primeiro provider:** OpenCode Go\
**Backend previsto:** Go\
**Implantação:** Local-first

------------------------------------------------------------------------

# 1. Instruções para o agente de desenvolvimento

Este documento é a referência inicial para iniciar o projeto AI Hub. O
agente deve utilizá-lo como contexto de produto, arquitetura e
planejamento.

## 1.1 Objetivo imediato

Iniciar a implementação de **E01 --- Core Foundation** e **E02 ---
Provider Management**, respeitando a estratégia Gateway First.

O primeiro resultado técnico esperado é uma aplicação local em Go que
consiga:

1.  Inicializar de forma confiável.
2.  Carregar e validar configurações.
3.  Expor um servidor HTTP local.
4.  Disponibilizar health checks.
5.  Registrar logs estruturados e correlation IDs.
6.  Configurar o provider OpenCode Go.
7.  Proteger a credencial do provider.
8.  Verificar conectividade.
9.  Consultar modelos disponíveis, quando suportado.
10. Executar uma chamada de teste usando um modelo e protocolo
    compatíveis.

**Não implementar ainda** o motor completo de roteamento inteligente,
aprendizado, dashboard ou todos os harnesses. O objetivo inicial é
provar a fundação e a conectividade real.

## 1.2 Regras de trabalho

-   Antes de codificar, inspecionar o repositório existente. Não
    presumir que ele esteja vazio.
-   Se já houver decisões, código ou documentação no repositório,
    comparar com este documento e apontar conflitos antes de
    sobrescrever.
-   Trabalhar incrementalmente, com commits lógicos se o ambiente
    permitir.
-   Aplicar testes automatizados desde o início.
-   Não criar abstrações especulativas sem necessidade demonstrada pelo
    primeiro fluxo.
-   Não implementar microserviços. A arquitetura inicial é um **modular
    monolith**.
-   Não adicionar Redis, Kafka, Kubernetes ou infraestrutura distribuída
    sem requisito comprovado.
-   Não armazenar segredos em código, arquivos versionados, logs ou
    respostas HTTP.
-   Não declarar uma integração funcional sem executar testes reais
    contra o provider, quando houver credenciais disponíveis.
-   Diferenciar claramente funcionalidades implementadas, simuladas,
    estimadas e ainda não verificadas.
-   Quando uma decisão não estiver especificada, escolher a solução mais
    simples que preserve a evolução futura e registrar a decisão.
-   Atualizar os documentos e critérios de aceite à medida que a
    implementação revelar novas necessidades.
-   Não tentar antecipar todos os requisitos dos próximos épicos.

## 1.3 Processo recomendado

Para cada incremento:

1.  Inspecionar o estado atual do repositório.
2.  Identificar o escopo exato.
3.  Registrar dúvidas e riscos relevantes.
4.  Propor a implementação mínima.
5.  Definir ou refinar os critérios de aceite daquele incremento.
6.  Implementar.
7.  Executar testes unitários, integração e verificações estáticas
    aplicáveis.
8.  Validar manualmente o fluxo quando necessário.
9.  Atualizar documentação, backlog e decisões arquiteturais.
10. Informar o que foi concluído, o que não foi e quais dependências
    permanecem.

------------------------------------------------------------------------

# 2. Visão do produto

## 2.1 O que é o AI Hub

O **AI Hub** é um Personal AI Orchestrator: uma plataforma pessoal que
centraliza a conexão e a gestão de diferentes modelos de inteligência
artificial, provedores, assinaturas, franquias e ambientes de
desenvolvimento.

O usuário conecta seus harnesses a um gateway próprio. O AI Hub
administra a comunicação com os providers e, progressivamente, passa a
decidir qual modelo utilizar em cada tarefa, considerando qualidade,
capacidades, disponibilidade, limites, custos e políticas definidas pelo
usuário.

A proposta não é apenas oferecer um proxy de LLM. É construir uma camada
de orquestração que administre os recursos de IA disponíveis ao usuário.

**Conceito central:** o usuário não quer administrar modelos de IA
manualmente. Quer administrar um sistema que administre os modelos por
ele.

## 2.2 Problema que queremos resolver

O usuário utiliza diferentes ferramentas e serviços, cada um com seus
próprios modelos, credenciais, limites, créditos, franquias e formas de
integração.

Entre as ferramentas prioritárias estão:

-   OpenCode.
-   OpenCode Go.
-   OpenRouter.
-   OpenAI Codex.
-   Claude / Claude Code.
-   Cursor.

O usuário não quer:

-   Escolher manualmente um modelo para cada tarefa.
-   Alternar entre providers e configurações em vários lugares.
-   Monitorar limites em vários painéis.
-   Consumir créditos pagos quando há recursos gratuitos ou franquias
    adequadas.
-   Reconfigurar cada harness sempre que adiciona um provider.
-   Utilizar modelos premium em tarefas simples quando um modelo
    gratuito resolve satisfatoriamente.

## 2.3 Proposta de valor

O AI Hub deverá oferecer:

-   Um ponto central de conexão para diferentes harnesses.
-   Independência em relação a um único provider.
-   Catálogo centralizado de modelos e capacidades.
-   Seleção de modelos conforme tarefa e políticas.
-   Prioridade a modelos gratuitos quando forem adequados.
-   Uso seletivo de recursos premium ou franquias de assinatura.
-   Gestão centralizada de limites, custos e consumo.
-   Aprendizado contínuo a partir dos resultados das execuções.
-   Transparência sobre as decisões tomadas.

Tagline conceitual:

> One connection. Every model. Intelligent execution.

------------------------------------------------------------------------

# 3. Princípios do produto

## 3.1 User-driven

O usuário define políticas, permissões, limites e nível de autonomia. O
sistema não deve impor um único modo de operação.

## 3.2 Autonomia configurável

O nível de automação será configurável pelo usuário. Os níveis
conceituais são:

1.  **Manual:** o usuário escolhe modelo e provider.
2.  **Assistido:** o sistema recomenda e o usuário confirma.
3.  **Semiautônomo:** o sistema escolhe, mas solicita confirmação para
    ações ou condições configuradas.
4.  **Autônomo:** o sistema decide e executa dentro das políticas
    autorizadas.

A configuração de autonomia deve ser independente de políticas
financeiras e de segurança. Mesmo no modo autônomo, o sistema deve
respeitar limites, permissões e restrições explícitas.

## 3.3 Free-first

A política padrão pretendida é priorizar a melhor LLM gratuita adequada
a cada tipo de tarefa.

O objetivo não é simplesmente escolher o modelo mais barato. É escolher
o melhor modelo disponível dentro das políticas e recursos do usuário.

## 3.4 Quality-driven

A qualidade real para as tarefas do usuário deve orientar as decisões.
Rankings e benchmarks externos podem ser referências, mas não substituem
avaliações próprias.

## 3.5 Premium seletivo

O usuário poderá permitir o uso de recursos premium ou franquias
associadas a serviços como Cursor, Codex e Claude. A utilização deverá
depender de políticas configuráveis e de evidências de que o recurso
oferece vantagem relevante.

Não presumir que uma assinatura permite consumo por API externa ou por
gateway personalizado. Cada integração precisa ter suas possibilidades
técnicas e contratuais verificadas.

## 3.6 Learning-first

O sistema deve aprender continuamente com os resultados das execuções.

O aprendizado deverá considerar, conforme os dados disponíveis:

-   Tipo e complexidade da tarefa.
-   Modelo e provider utilizados.
-   Tempo de execução.
-   Consumo e custo informado.
-   Conclusão ou falha.
-   Erros, retries e fallback.
-   Avaliação do resultado.
-   Contexto de execução e capacidades utilizadas.

O sistema não deve concluir que um modelo é superior com base em poucas
amostras ou em sinais frágeis. A avaliação deve considerar quantidade e
qualidade das evidências.

## 3.7 Provider-agnostic

O domínio central não deve depender de um fornecedor específico. Cada
provider deve ser integrado por adapter.

## 3.8 Harness-agnostic

O núcleo deve suportar diferentes harnesses e protocolos, preservando os
recursos compatíveis com cada integração.

## 3.9 Cost-aware

Toda decisão deve considerar as políticas financeiras do usuário. Gastos
não autorizados devem ser bloqueados, mesmo em modo autônomo.

## 3.10 Transparent

O usuário deve conseguir entender por que um modelo foi escolhido, quais
políticas foram aplicadas e quais informações fundamentaram a decisão.

------------------------------------------------------------------------

# 4. Harnesses e providers

É fundamental distinguir quatro conceitos:

-   **Harness:** ambiente que executa o trabalho e interage com o
    usuário.
-   **Provider:** serviço que disponibiliza acesso a modelos.
-   **Model:** modelo de linguagem efetivamente utilizado.
-   **Gateway:** ponto central de entrada e orquestração.

Exemplo conceitual:

``` text
Codex (Harness)
    |
    v
AI Hub (Gateway)
    |
    v
Decision Engine
    |
    v
OpenCode Go (Provider)
    |
    v
DeepSeek (Model)
```

## 4.1 Harnesses prioritários

As integrações-alvo são:

1.  OpenCode.
2.  Codex.
3.  Claude / Claude Code.
4.  Cursor.

Esses são os ambientes de maior interesse do usuário. A ordem efetiva de
implementação poderá ser ajustada conforme viabilidade e
compatibilidade.

## 4.2 Providers prioritários

Entre os providers e ecossistemas considerados:

-   OpenCode Go.
-   OpenRouter.
-   Anthropic.
-   OpenAI.
-   Outros providers que venham a ser adicionados.

Não confundir o nome de um harness com o provider utilizado por ele. Por
exemplo, OpenCode pode funcionar como harness, enquanto OpenCode Go é um
serviço/provider de modelos.

## 4.3 Limites de integração

Nem todos os harnesses permitem configurar um endpoint externo ou
utilizar a franquia de uma assinatura fora do próprio produto.

O agente deve verificar a documentação oficial e testar cada integração.
Não deve presumir que:

-   Toda assinatura oferece uma API pública.
-   Toda franquia pode ser consumida por um gateway externo.
-   Todos os providers suportam os mesmos protocolos.
-   Todos os modelos de um provider suportam as mesmas capacidades.
-   Uma compatibilidade OpenAI genérica preserva automaticamente
    recursos específicos de cada harness.

------------------------------------------------------------------------

# 5. Modos de operação

## 5.1 Manual

O usuário escolhe explicitamente o modelo e o provider. O AI Hub
continua centralizando políticas, limites, consumo, custos e registros.

## 5.2 Assistido

O sistema analisa a solicitação e apresenta uma recomendação. O usuário
confirma antes da execução, conforme configuração.

## 5.3 Semiautônomo

O sistema escolhe e executa automaticamente dentro das permissões
concedidas, mas solicita confirmação para situações configuradas, como
uso de recursos pagos ou excedentes.

## 5.4 Autônomo

O sistema classifica a tarefa, escolhe o recurso, executa, monitora e
realiza fallback quando permitido. Todas as decisões continuam limitadas
pelas políticas do usuário.

------------------------------------------------------------------------

# 6. Inteligência do produto

A inteligência do AI Hub será desenvolvida progressivamente.

## 6.1 Task Intelligence

Responsável por entender a natureza e a complexidade da tarefa.

Categorias iniciais possíveis:

-   Plan.
-   Coding.
-   Debug.
-   Review.
-   Research.
-   Chat.
-   Documentation.
-   Fast / Simple tasks.

Essa lista é inicial e poderá evoluir durante o uso.

## 6.2 Resource Intelligence

Responsável por conhecer os recursos disponíveis:

-   Providers cadastrados.
-   Modelos disponíveis.
-   Capacidades e protocolos.
-   Disponibilidade.
-   Limites e franquias.
-   Custos.
-   Velocidade e latência.
-   Evidências históricas de qualidade.

Os dados devem ter proveniência clara: valor confirmado por API,
configuração manual, estimativa ou informação desatualizada.

## 6.3 Decision Engine

Combina:

-   Tipo e complexidade da tarefa.
-   Capacidades exigidas.
-   Qualidade histórica.
-   Disponibilidade.
-   Políticas do usuário.
-   Franquias e limites.
-   Custo permitido.
-   Preferências de autonomia.

O resultado é uma decisão de execução justificável e rastreável.

## 6.4 Roteamento híbrido

A estratégia pretendida é híbrida:

1.  Aplicar regras determinísticas quando forem suficientes.
2.  Acionar uma LLM classificadora quando houver ambiguidade ou
    necessidade de interpretação.
3.  Combinar a classificação com recursos, avaliações e políticas no
    Decision Engine.

A LLM classificadora não pode ignorar políticas de segurança ou limites
financeiros.

## 6.5 Learning Engine

O sistema deverá registrar resultados e melhorar progressivamente suas
decisões.

Exemplos de sinais:

-   Sucesso ou falha.
-   Avaliação explícita do usuário.
-   Tempo de resposta.
-   Consumo.
-   Qualidade do resultado.
-   Necessidade de correção ou nova tentativa.
-   Adequação do modelo à tarefa.

O mecanismo de avaliação e aprendizado será detalhado durante os épicos
correspondentes. Não é escopo do primeiro incremento.

------------------------------------------------------------------------

# 7. Arquitetura de alto nível

## 7.1 Decisões arquiteturais iniciais

  Tema              Direção
  ----------------- -------------------------------------------------
  Implantação       Local-first
  Arquitetura       Modular Monolith
  Backend           Go
  Dashboard         React + Vite
  Integração        Protocolos compatíveis + adapters específicos
  Roteamento        Híbrido: regras + LLM quando necessário
  Execução          Híbrida, síncrona ou assíncrona conforme tarefa
  Persistência      Tecnologia a definir no desenho detalhado
  Observabilidade   OpenTelemetry como direção inicial
  Streaming         Preservar streaming quando suportado
  Distribuição      Aplicação local; Docker ou binário a decidir

## 7.2 Local-first

O AI Hub deverá funcionar prioritariamente na máquina do usuário.

Características esperadas:

-   Credenciais protegidas localmente.
-   Gateway acessível localmente.
-   Dashboard acessível pelo navegador.
-   Persistência local.
-   Comunicação direta com providers externos.
-   Sem dependência obrigatória de infraestrutura cloud própria.

Local-first não significa local-only. Backup, sincronização ou acesso
remoto poderão ser considerados futuramente, mas não são requisitos
iniciais.

## 7.3 Modular Monolith

A aplicação começará como um único sistema modular em Go.

Não criar microserviços prematuramente. Os limites internos devem ser
claros, mas a implantação inicial deve permanecer simples.

## 7.4 Camadas conceituais

### Integration Layer

Recebe solicitações dos harnesses e expõe protocolos compatíveis.

Responsabilidades:

-   HTTP APIs.
-   Adapters de protocolo.
-   Autenticação do gateway.
-   Normalização de requisições.
-   Streaming.

### Orchestration Layer

Coordena o ciclo de vida das solicitações e execuções.

Responsabilidades futuras:

-   Request Manager.
-   Execution Manager.
-   Session Context.
-   Fallback.

### Intelligence Layer

Entende tarefas, toma decisões e aprende.

Responsabilidades futuras:

-   Task Classifier.
-   Decision Engine.
-   Model Evaluator.
-   Learning Engine.

### Resource Layer

Gerencia provedores, modelos e políticas.

Responsabilidades:

-   Provider Registry.
-   Model Registry.
-   Quota Manager.
-   Policy Engine.
-   Provider Adapters.

### Infrastructure Layer

Oferece suporte transversal.

Responsabilidades:

-   Storage.
-   Job Scheduler.
-   Event Processing.
-   Telemetry.
-   Secrets.

O Dashboard será um cliente das APIs do sistema. Não deverá duplicar
regras de negócio.

## 7.5 Fluxo conceitual futuro

``` text
Harness
   |
   v
Integration Layer
   |
   v
Orchestration Layer
   |
   +----> Task Intelligence
   |
   +----> Decision Engine
   |          |
   |          +----> Policy Engine
   |          |
   |          +----> Resource Manager
   |
   v
Execution
   |
   v
Provider Adapter
   |
   v
LLM Provider
   |
   v
Response / Stream
   |
   +----> Usage Tracking
   |
   +----> Evaluation
   |
   +----> Learning Engine
```

Esse é o fluxo-alvo do produto. O primeiro incremento implementará
apenas a parte necessária para conectividade.

------------------------------------------------------------------------

# 8. Execução, estado e persistência

## 8.1 Execução híbrida

Nem toda operação precisa ser executada da mesma maneira.

Estratégia conceitual:

  Operação                  Estratégia pretendida
  ------------------------- -----------------------------------------
  Chat                      Síncrona com streaming
  Coding                    Síncrona com streaming quando suportado
  Classificação             Síncrona sob demanda
  Benchmark                 Assíncrona
  Avaliação de resultados   Assíncrona
  Aprendizado               Assíncrono
  Atualização de catálogo   Assíncrona
  Relatórios                Assíncrona

## 8.2 Stateless gateway e estado persistido

O gateway não deve depender da memória local do processo para recuperar
uma execução ou sessão.

Isso não significa que o sistema será totalmente sem estado.
Configurações, histórico, políticas, métricas e avaliações precisarão
ser persistidos.

## 8.3 Persistência

A tecnologia ainda não foi escolhida.

A decisão deverá considerar:

-   Execução local.
-   Simplicidade operacional.
-   Concorrência.
-   Transações.
-   Histórico de execuções.
-   Consultas analíticas.
-   Evolução do Learning Engine.
-   Backup e recuperação.

SQLite e PostgreSQL são alternativas a avaliar, não decisões já tomadas.

------------------------------------------------------------------------

# 9. Plano de implementação --- Gateway First

A estratégia escolhida é **Gateway First**.

A primeira prioridade é estabelecer uma infraestrutura confiável que
receba solicitações, comunique-se com providers e devolva respostas aos
harnesses.

A inteligência de roteamento será construída sobre esse núcleo
posteriormente.

## 9.1 Épicos propostos

  -----------------------------------------------------------------------
  Épico                   Nome                    Responsabilidade
  ----------------------- ----------------------- -----------------------
  E01                     Core Foundation         Inicialização,
                                                  configuração, servidor
                                                  HTTP e estrutura
                                                  modular

  E02                     Provider Management     Cadastro, credenciais,
                                                  adapters e
                                                  conectividade

  E03                     Model Catalog           Catálogo, capacidades e
                                                  descoberta de modelos

  E04                     API Gateway             Endpoints compatíveis,
                                                  autenticação,
                                                  normalização

  E05                     Execution & Streaming   Execução, streaming,
                                                  erros, timeouts e
                                                  fallback

  E06                     Usage & Observability   Histórico, métricas,
                                                  consumo e
                                                  rastreabilidade

  E07                     First Harness           Integração real com o
                          Integration             primeiro harness

  E08                     Intelligent Routing     Classificação e seleção
                                                  automática

  E09                     Resource Management     Franquias, limites,
                                                  custos e políticas

  E10                     Learning Engine         Avaliação e aprendizado
                                                  contínuo

  E11                     Dashboard               Configuração,
                                                  monitoramento e
                                                  gerenciamento
  -----------------------------------------------------------------------

A lista é um planejamento inicial. A ordem e a granularidade poderão
mudar conforme dependências reais.

## 9.2 Marcos

  -----------------------------------------------------------------------
  Marco                               Resultado
  ----------------------------------- -----------------------------------
  M1 --- Connectivity                 Gateway comunica-se com um provider

  M2 --- Compatibility                Um harness utiliza o gateway

  M3 --- Reliability                  Streaming, erros, rastreabilidade e
                                      testes consolidados
  -----------------------------------------------------------------------

## 9.3 Primeiro incremento

O primeiro incremento deve provar a fundação e a integração:

``` text
Codex
   |
   v
AI Hub local
   |
   v
Provider Adapter
   |
   v
OpenCode Go
   |
   v
Modelo compatível
   |
   v
Resposta ao Codex
```

A primeira integração escolhida é:

-   Harness: OpenAI Codex.
-   Provider: OpenCode Go.
-   Modelo candidato: DeepSeek, sujeito à confirmação de disponibilidade
    e compatibilidade.

------------------------------------------------------------------------

# 10. E01 --- Core Foundation

## 10.1 Objetivo

Estabelecer a fundação executável do AI Hub, com configuração,
inicialização, servidor HTTP, estrutura modular e encerramento
controlado.

## 10.2 Capacidades

  -----------------------------------------------------------------------
  ID                      Capacidade              Descrição
  ----------------------- ----------------------- -----------------------
  E01-C01                 Application Bootstrap   Inicialização e
                                                  encerramento
                                                  controlados

  E01-C02                 Configuration           Carregamento e
                                                  validação de
                                                  configurações

  E01-C03                 HTTP Server             Servidor HTTP local

  E01-C04                 Module Management       Organização e
                                                  inicialização dos
                                                  módulos

  E01-C05                 Health Monitoring       Verificação da saúde da
                                                  aplicação

  E01-C06                 Logging                 Logs estruturados e
                                                  rastreáveis
  -----------------------------------------------------------------------

## 10.3 Requisitos funcionais iniciais

-   RF-001: o sistema deve iniciar como aplicação local.
-   RF-002: o sistema deve carregar suas configurações de inicialização.
-   RF-003: configurações inválidas devem impedir inicialização
    inconsistente.
-   RF-004: o servidor HTTP deve disponibilizar endpoints de saúde.
-   RF-005: o sistema deve permitir encerramento gracioso.
-   RF-006: cada requisição deve possuir identificador de correlação.
-   RF-007: falhas de inicialização devem apresentar informações úteis
    para diagnóstico.

## 10.4 Critérios de aceite iniciais

### AC-001 --- Inicialização

**Given** que as configurações são válidas,\
**When** o usuário iniciar o AI Hub,\
**Then** o servidor deverá inicializar e disponibilizar seu endpoint de
saúde.

### AC-002 --- Configuração inválida

**Given** que existe uma configuração obrigatória inválida,\
**When** a aplicação iniciar,\
**Then** deverá informar o erro e interromper a inicialização.

### AC-003 --- Health check

**Given** que o servidor está em execução,\
**When** o cliente consultar o endpoint de saúde,\
**Then** deverá receber uma resposta indicando a disponibilidade da
aplicação.

### AC-004 --- Graceful shutdown

**Given** que a aplicação está atendendo requisições,\
**When** receber um sinal de encerramento,\
**Then** deverá interromper novas requisições e finalizar as operações
em andamento dentro do limite configurado.

### AC-005 --- Correlation ID

**Given** uma requisição HTTP recebida,\
**When** ela atravessar o servidor,\
**Then** deverá possuir um identificador de correlação disponível nos
logs.

## 10.5 Estrutura inicial sugerida

``` text
ai-hub/
├── cmd/
│   └── ai-hub/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── server/
│   ├── health/
│   └── observability/
├── pkg/
├── configs/
├── docs/
├── go.mod
└── README.md
```

Não criar pastas vazias sem necessidade. A estrutura representa
responsabilidades previstas e deverá ser adaptada ao código real.

------------------------------------------------------------------------

# 11. E02 --- Provider Management

## 11.1 Objetivo

Criar a capacidade de integrar, configurar e administrar providers de
LLM sem acoplar o núcleo do AI Hub a um fornecedor específico.

O primeiro provider será o OpenCode Go.

## 11.2 Capacidades

  -----------------------------------------------------------------------
  ID                      Capacidade              Descrição
  ----------------------- ----------------------- -----------------------
  E02-C01                 Provider Registry       Registrar provedores

  E02-C02                 Provider Configuration  Configurar endpoints e
                                                  parâmetros

  E02-C03                 Credential Management   Gerenciar credenciais
                                                  com segurança

  E02-C04                 Provider Adapter        Definir contrato de
                                                  integração

  E02-C05                 Provider Connectivity   Verificar conectividade

  E02-C06                 Provider Lifecycle      Habilitar, desabilitar
                                                  e atualizar provedores
  -----------------------------------------------------------------------

## 11.3 Requisitos funcionais iniciais

-   RF-008: o sistema deve permitir cadastrar um provider.
-   RF-009: deve permitir configurar nome, endpoint, protocolo e
    credencial.
-   RF-010: deve proteger credenciais armazenadas localmente.
-   RF-011: deve implementar adapters independentes por provider.
-   RF-012: deve permitir testar a conectividade de um provider.
-   RF-013: deve identificar os protocolos suportados pelo provider.
-   RF-014: deve permitir habilitar e desabilitar um provider.
-   RF-015: deve registrar erros de integração sem expor credenciais.

## 11.4 Contrato inicial de provider

Proposta conceitual em Go:

``` go
type Provider interface {
    ID() string

    Capabilities() ProviderCapabilities

    ListModels(
        ctx context.Context,
    ) ([]Model, error)

    Execute(
        ctx context.Context,
        request Request,
    ) (Response, error)

    Stream(
        ctx context.Context,
        request Request,
    ) (Stream, error)
}
```

Esse contrato não deve ser considerado definitivo antes de detalhar os
contratos de protocolo e execução.

Princípios:

-   `Request` e `Response` devem ser tipos internos normalizados.
-   O adapter converte o contrato interno para o protocolo externo.
-   Capacidades devem ser declaradas explicitamente.
-   Não obrigar todos os providers a implementar funcionalidades que não
    suportam.
-   Não espalhar particularidades do OpenCode Go pelo domínio central.
-   Preservar extensibilidade sem criar abstrações desnecessárias.

## 11.5 Configuração inicial do OpenCode Go

Referência inicial de configuração:

  Campo                Valor
  -------------------- ---------------------------------------------
  Provider ID          `opencode-go`
  Nome                 OpenCode Go
  Base URL             A confirmar na documentação oficial vigente
  Protocolo desejado   Responses
  Autenticação         API Key
  Credencial           `OPENCODE_API_KEY`

**Atenção:** não assumir que endpoint, protocolo e disponibilidade de
modelo estão confirmados. O agente deve verificar a documentação oficial
atual do OpenCode Go e validar a configuração com uma chamada real.

O modelo DeepSeek é o candidato inicial do usuário. Sua disponibilidade
e compatibilidade com Responses devem ser comprovadas antes de declarar
a integração concluída.

## 11.6 Critérios de aceite iniciais

### AC-006 --- Cadastro de provider

**Given** uma configuração válida,\
**When** o usuário cadastrar um provider,\
**Then** o sistema deverá registrá-lo e disponibilizá-lo para
utilização.

### AC-007 --- Proteção de credenciais

**Given** uma API Key cadastrada,\
**When** o sistema persistir ou registrar informações,\
**Then** a credencial não poderá aparecer em logs ou respostas da API e
deverá ser protegida em armazenamento local.

### AC-008 --- Conectividade

**Given** um provider configurado,\
**When** o usuário solicitar um teste de conexão,\
**Then** o sistema deverá informar se conseguiu estabelecer comunicação
e identificar a natureza de eventuais falhas.

### AC-009 --- Capacidades

**Given** um provider conectado,\
**When** o sistema consultar suas capacidades,\
**Then** deverá identificar os protocolos e operações efetivamente
suportados.

### AC-010 --- Isolamento

**Given** dois providers implementados,\
**When** uma integração for modificada,\
**Then** a implementação de outro provider não deverá precisar ser
alterada, salvo mudanças em contratos compartilhados.

### AC-011 --- Desativação

**Given** um provider cadastrado,\
**When** o usuário desativá-lo,\
**Then** novas execuções não deverão utilizar esse provider.

------------------------------------------------------------------------

# 12. Integração Codex + OpenCode Go

## 12.1 Fluxo desejado

``` text
Codex
  |
  | OpenAI Responses API
  v
AI Hub — Responses Gateway
  |
  | Contrato interno normalizado
  v
OpenCode Go Adapter
  |
  | Protocolo upstream validado
  v
OpenCode Go
  |
  v
Modelo selecionado
  |
  v
Resposta / Stream
  |
  v
AI Hub
  |
  v
Codex
```

## 12.2 Compatibilidade de protocolo

O Codex utiliza o protocolo Responses para providers personalizados na
configuração considerada neste projeto.

O OpenCode Go oferece diferentes possibilidades de integração. É
necessário verificar a documentação vigente e a compatibilidade
específica do modelo escolhido.

A primeira implementação deverá preferir comunicação
Responses-to-Responses, caso seja suportada pelo endpoint e modelo
selecionados.

Não implementar tradução entre Chat Completions e Responses neste
primeiro incremento sem necessidade comprovada. Uma tradução pode perder
recursos, metadados ou semântica.

## 12.3 Recursos a validar

-   Autenticação.
-   Identificação de modelo.
-   Formato de mensagens e instruções.
-   Streaming.
-   Finalização de resposta.
-   Erros e códigos HTTP.
-   Timeouts e cancelamento.
-   Usage / tokens, quando informado.
-   Tool calling, quando necessário.
-   Reasoning e metadados, quando suportados.
-   Limites de contexto.
-   Compatibilidade do modelo com o endpoint.

Não declarar compatibilidade total apenas porque uma chamada simples de
texto funcionou.

## 12.4 Teste de integração real

O teste deve:

1.  Carregar a credencial de forma segura.
2.  Consultar modelos disponíveis, se o provider oferecer endpoint de
    catálogo.
3.  Selecionar um modelo comprovadamente compatível.
4.  Enviar uma requisição mínima.
5.  Receber uma resposta.
6.  Testar streaming.
7.  Validar erros de autenticação e modelo inválido.
8.  Confirmar que a credencial não aparece nos logs.

Se não houver credencial disponível no ambiente de desenvolvimento,
criar testes com servidor HTTP simulado e deixar explícito que a
integração real ainda não foi validada.

------------------------------------------------------------------------

# 13. Próximos épicos --- fora do escopo imediato

Os seguintes temas fazem parte da evolução do produto, mas não devem
bloquear E01 e E02:

## E03 --- Model Catalog

-   Registro e descoberta de modelos.
-   Capacidades.
-   Protocolos suportados.
-   Context window.
-   Modalidades de entrada e saída.
-   Associação entre modelo e provider.

## E04 --- API Gateway

-   Endpoints compatíveis.
-   Autenticação do gateway.
-   Normalização de requisições.
-   Respostas.
-   Streaming.
-   Compatibilidade com Codex.

## E05 --- Execution & Streaming

-   Ciclo de execução.
-   Timeouts.
-   Cancelamento.
-   Tratamento de erros.
-   Retries controlados.
-   Fallback.
-   Preservação de semântica do protocolo.

## E06 --- Usage & Observability

-   Histórico de chamadas.
-   Tokens e consumo informado.
-   Custos quando disponíveis.
-   Latência.
-   Erros.
-   Rastreabilidade.

## E07 --- First Harness Integration

-   Configuração do Codex para apontar ao AI Hub.
-   Testes reais de utilização.
-   Validação de recursos do harness.
-   Documentação de configuração.

## E08 --- Intelligent Routing

-   Classificação de tarefas.
-   Políticas de roteamento.
-   Seleção de modelos.
-   Decisões explicáveis.

## E09 --- Resource Management

-   Franquias.
-   Quotas.
-   Créditos.
-   Custos.
-   Limites.
-   Permissões.

## E10 --- Learning Engine

-   Avaliação de resultados.
-   Histórico de desempenho.
-   Comparação de modelos.
-   Aprendizado contínuo.

## E11 --- Dashboard

-   Configurações.
-   Providers.
-   Modelos.
-   Políticas.
-   Histórico.
-   Métricas.
-   Controles de autonomia.

------------------------------------------------------------------------

# 14. Riscos e cuidados arquiteturais

## 14.1 Quotas e franquias

Nem todo provider disponibiliza APIs para consultar franquias de
assinatura. O sistema deve distinguir dados oficiais, configuração
manual e estimativas.

## 14.2 Assinaturas e acesso externo

Não assumir que Cursor, Codex ou Claude permitem utilizar suas franquias
através de um gateway externo. Verificar capacidades e condições de cada
produto.

## 14.3 Streaming e fallback

Não trocar de modelo no meio de um stream sem uma estratégia explícita.
A primeira decisão de fallback deve ocorrer antes de emitir conteúdo ao
cliente, salvo quando houver mecanismo seguro e definido para
continuidade.

## 14.4 Sessões e contexto

A continuidade entre modelos pode exigir afinidade de sessão, histórico
ou transformação de contexto. Não assumir que providers diferentes
aceitam exatamente o mesmo contexto.

## 14.5 Segurança

-   Não expor credenciais.
-   Não registrar conteúdo sensível sem necessidade.
-   Proteger o endpoint local contra acesso indevido na rede.
-   Definir autenticação do gateway antes de expor o serviço além de
    localhost.
-   Aplicar timeouts e limites de payload.
-   Validar endpoints e certificados TLS.
-   Não permitir que uma classificação de IA contorne políticas de
    segurança ou orçamento.

## 14.6 Observabilidade e proveniência

Todo dado de consumo, quota, custo ou desempenho deverá indicar sua
origem e confiabilidade. Não apresentar estimativas como dados oficiais.

------------------------------------------------------------------------

# 15. Decisões arquiteturais iniciais (ADRs)

As decisões abaixo representam a direção aprovada até o momento.
Detalhes poderão ser refinados por ADRs específicos.

  ADR       Decisão
  --------- ------------------------------------------------------------
  ADR-001   Local-first deployment
  ADR-002   Modular monolith
  ADR-003   Separação entre harness, gateway, provider e model
  ADR-004   Compatibilidade por protocolo + adapters específicos
  ADR-005   Roteamento híbrido: regras + LLM quando necessário
  ADR-006   Execução síncrona e assíncrona conforme a tarefa
  ADR-007   Tecnologia de persistência adiada para desenho detalhado
  ADR-008   Gateway First como estratégia de implementação
  ADR-009   Codex + OpenCode Go como primeira combinação de integração

------------------------------------------------------------------------

# 16. Definition of Done inicial

Um incremento só deve ser considerado concluído quando:

-   O escopo definido estiver implementado.
-   Os critérios de aceite aplicáveis forem atendidos.
-   Testes automatizados relevantes passarem.
-   Erros e casos negativos importantes forem cobertos.
-   Logs não expuserem credenciais.
-   A documentação estiver atualizada.
-   Limitações conhecidas estiverem registradas.
-   A validação real estiver identificada como realizada ou pendente.
-   Não houver funcionalidades simuladas apresentadas como concluídas.

A Definition of Done poderá evoluir com o projeto.

------------------------------------------------------------------------

# 17. Plano de execução imediato para o agente

O agente deve seguir esta sequência:

## Passo 1 --- Inspecionar o ambiente

-   Verificar se existe repositório e código.
-   Identificar linguagem, estrutura, ferramentas e convenções
    existentes.
-   Verificar arquivos de documentação e instruções do projeto.
-   Não sobrescrever arquivos existentes sem analisar seu conteúdo.

## Passo 2 --- Preparar a fundação E01

-   Definir bootstrap da aplicação.
-   Implementar configuração e validação.
-   Implementar servidor HTTP local.
-   Implementar health check.
-   Implementar logs estruturados.
-   Implementar correlation ID.
-   Implementar graceful shutdown.
-   Criar testes.

## Passo 3 --- Implementar E02 de forma incremental

-   Definir os conceitos mínimos de Provider e ProviderCapabilities.
-   Criar registro de providers.
-   Criar configuração do OpenCode Go.
-   Implementar gerenciamento seguro da credencial.
-   Criar adapter do OpenCode Go.
-   Implementar teste de conectividade.
-   Implementar consulta de modelos se suportada.
-   Criar testes unitários e de integração.

## Passo 4 --- Validar protocolo e modelo

-   Consultar documentação oficial atual.
-   Confirmar endpoint upstream.
-   Confirmar suporte ao protocolo Responses.
-   Confirmar disponibilidade do modelo DeepSeek candidato.
-   Executar chamada real quando houver credenciais.
-   Testar streaming.
-   Documentar limitações e incompatibilidades.

## Passo 5 --- Encerrar o incremento

Apresentar:

-   O que foi implementado.
-   Estrutura final do código.
-   Decisões técnicas tomadas.
-   Testes executados e resultados.
-   Validações reais realizadas.
-   Critérios de aceite atendidos ou pendentes.
-   Riscos e próximos passos.

Não iniciar E03 automaticamente sem revisar os resultados de E01 e E02.

------------------------------------------------------------------------

# 18. O que não fazer agora

Para manter foco e evitar sobreengenharia, não implementar nesta etapa:

-   Classificador de tarefas com LLM.
-   Decision Engine completo.
-   Learning Engine.
-   Benchmark automatizado de modelos.
-   Gerenciamento avançado de quotas.
-   Integrações simultâneas com todos os harnesses.
-   Dashboard completo.
-   Sistema distribuído.
-   Microserviços.
-   Kubernetes.
-   Filas externas sem necessidade comprovada.
-   Sincronização cloud.
-   Troca de modelo no meio de uma resposta em streaming.

Esses itens pertencem a incrementos posteriores.

------------------------------------------------------------------------

# 19. Resultado esperado do primeiro ciclo

Ao concluir E01 e E02, esperamos ter:

1.  Uma aplicação Go local e funcional.
2.  Configuração validada e inicialização controlada.
3.  Servidor HTTP com health checks.
4.  Logs estruturados e rastreabilidade.
5.  Uma abstração inicial de providers.
6.  Credencial OpenCode Go protegida.
7.  Adapter OpenCode Go.
8.  Teste de conectividade.
9.  Catálogo de modelos consultado, caso suportado.
10. Evidência documentada da compatibilidade do modelo candidato com o
    protocolo escolhido.

O resultado ainda não será o AI Hub completo. Será a fundação técnica
comprovada sobre a qual construiremos o gateway, a execução compatível
com Codex e, posteriormente, a inteligência de roteamento.

------------------------------------------------------------------------

# 20. Orientação final ao agente

Trate este documento como a baseline inicial de produto e arquitetura,
não como uma especificação imutável.

O projeto deve evoluir por incrementos pequenos. Especifique o
suficiente para implementar o próximo incremento com segurança, valide
com testes e uso real, e atualize os documentos quando novas evidências
exigirem mudanças.

**Primeira missão: inspecionar o repositório e iniciar E01 --- Core
Foundation.**

Não comece construindo todo o AI Hub. Primeiro prove a fundação. Em
seguida implemente E02 e valide a integração real com OpenCode Go.

A prioridade é entregar uma base simples, confiável, testável e
extensível.
