import { useRef } from "react";
import { Download, Upload, Sliders, Layers, HardDrive } from "lucide-react";

interface HeaderProps {
  onExport: () => void;
  onImport: (file: File) => void;
  onOpenManage: () => void;
  onLinkSSD: () => void;
  isLinkedToFile: boolean;
  linkedFileName: string | null;
}

export function Header({
  onExport,
  onImport,
  onOpenManage,
  onLinkSSD,
  isLinkedToFile,
  linkedFileName,
}: HeaderProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      onImport(file);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  return (
    <header className="flex flex-wrap items-center justify-between gap-4 border-b border-neutral-800 pb-4">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-900 border border-neutral-800 text-neutral-200">
          <Layers className="h-5 w-5 text-amber-500" />
        </div>
        <div>
          <h1 className="text-lg font-semibold tracking-tight text-neutral-100">
            Fila Dinâmica
          </h1>
          <p className="text-xs text-neutral-500">
            Execução flexível por prioridade e contexto
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {/* Botão de Conexão com o Arquivo do SSD */}
        <button
          onClick={onLinkSSD}
          title={
            isLinkedToFile
              ? `Gravando diretamente em: ${linkedFileName}`
              : "Clique para selecionar o arquivo routine_state.json no SSD compartilhado"
          }
          className={`flex items-center gap-2 rounded-lg border px-3 py-1.5 text-xs font-medium transition-all ${
            isLinkedToFile
              ? "border-emerald-800/80 bg-emerald-950/40 text-emerald-400"
              : "border-neutral-800 bg-neutral-900 text-neutral-300 hover:bg-neutral-800 hover:text-white"
          }`}
        >
          <HardDrive
            className={`h-3.5 w-3.5 ${isLinkedToFile ? "text-emerald-400" : "text-neutral-400"}`}
          />
          <span>{isLinkedToFile ? linkedFileName : "Conectar SSD"}</span>
          {isLinkedToFile && (
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
          )}
        </button>

        <input
          ref={fileInputRef}
          type="file"
          accept=".json"
          onChange={handleFileChange}
          className="hidden"
        />

        <button
          onClick={() => fileInputRef.current?.click()}
          title="Importar JSON manualmente"
          className="flex items-center gap-1.5 rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs font-medium text-neutral-300 hover:bg-neutral-800 hover:text-white transition-colors"
        >
          <Upload className="h-3.5 w-3.5" />
          <span>Importar</span>
        </button>

        <button
          onClick={onExport}
          title="Baixar cópia de segurança em JSON"
          className="flex items-center gap-1.5 rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs font-medium text-neutral-300 hover:bg-neutral-800 hover:text-white transition-colors"
        >
          <Download className="h-3.5 w-3.5" />
          <span>Exportar</span>
        </button>

        <button
          onClick={onOpenManage}
          title="Gerenciar Atividades e Regras"
          className="flex items-center gap-1.5 rounded-lg border border-neutral-700 bg-neutral-800 px-3 py-1.5 text-xs font-medium text-neutral-100 hover:bg-neutral-700 transition-colors"
        >
          <Sliders className="h-3.5 w-3.5 text-amber-400" />
          <span>Configurar</span>
        </button>
      </div>
    </header>
  );
}
