import { Link } from 'react-router-dom'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { Button } from '../../../shared/ui/button'

export function IntelligenceHelpPage() {
  return (
    <div className={appPageClassName('space-y-4', 'py-3', 'pb-8')}>
      <AppPageHeader
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link to="/intelligence">Voltar</Link>
          </Button>
        }
      />

      <article className="space-y-4 text-[0.8125rem] leading-relaxed text-secondary">
        <p className="rounded-[10px] border border-line bg-surface px-4 py-3 text-primary">
          O roteamento de inteligência escolhe um par provedor + modelo, ou recusa a chamada com um
          motivo. Ele não lê o texto da mensagem e não chama outro modelo para decidir.
        </p>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">Quem entra na disputa</h2>
          <p className="mt-2">Um modelo só compete se passar em tudo isto:</p>
          <ul className="mt-2 list-disc space-y-1.5 pl-5">
            <li>O provedor está ligado, tem credencial e o adaptador de respostas está ativo.</li>
            <li>O modelo existe no catálogo e o catálogo está fresco.</li>
            <li>
              Ele declara as capacidades que a requisição pede (ferramentas, imagem, JSON etc.).
              Capacidade desconhecida elimina o candidato.
            </li>
            <li>
              Se a requisição pede um limite de saída, esse limite cabe no que o catálogo declara.
            </li>
          </ul>
          <p className="mt-3">
            Saúde do provedor não desempata e não troca o alvo. Se sobrar mais de um candidato sem
            uma regra que escolha um só, a resposta é ambígua, não “o primeiro da lista”.
          </p>
        </section>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">Quem ganha</h2>
          <p className="mt-2">A ordem é fixa:</p>
          <ol className="mt-2 list-decimal space-y-2 pl-5">
            <li>
              <strong className="font-semibold text-primary">Provedor e modelo informados.</strong>{' '}
              Só aquele par. Se o modelo existir só em outro provedor, recusa. Não troca de
              provedor.
            </li>
            <li>
              <strong className="font-semibold text-primary">Só o provedor.</strong> Ganha se
              restar exatamente um modelo elegível nesse provedor. Zero ou vários: recusa.
            </li>
            <li>
              <strong className="font-semibold text-primary">Só o modelo.</strong> Ganha se existir
              exatamente um par elegível com esse id. Se o mesmo id aparecer em várias contas do
              mesmo tipo de provedor, o desempate é rodízio ponderado pela cota (AW-RR). Tipos
              diferentes de provedor continuam ambíguos.
            </li>
            <li>
              <strong className="font-semibold text-primary">Nada informado.</strong> Se o OpenCode
              Go tiver exatamente um modelo elegível, é ele. Senão, só escolhe se sobrar um único
              candidato no hub inteiro, com catálogos frescos. Vários candidatos: recusa.
            </li>
          </ol>
          <p className="mt-3">
            <code className="font-mono text-[0.75rem] text-primary">antigravity:nome</code> sem
            provedor explícito só fixa o provedor quando existe uma única conta Antigravity.
          </p>
        </section>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">aihub/auto</h2>
          <p className="mt-2">
            Esse id não é um modelo real. O fluxo é: lista permitida, depois só modelos com prova
            de contrato para o formato do pedido (cota, cooldown, smoke, schema), depois o pool com
            mais peso de cota, e dentro do pool a ordem administrativa. O perfil{' '}
            <code className="font-mono text-[0.75rem] text-primary">efficiency</code> empurra Gemini
            para o fim em chamadas curtas. O perfil{' '}
            <code className="font-mono text-[0.75rem] text-primary">intelligence</code> não muda a
            ordem; só registra que não há evidência extra.
          </p>
        </section>
      </article>
    </div>
  )
}
