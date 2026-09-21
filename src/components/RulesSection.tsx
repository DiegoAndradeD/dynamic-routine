import { useState } from "react";
import { ChevronDown, ChevronUp, AlertCircle, Compass } from "lucide-react";
import type { OperationalRule } from "../types";

interface RulesSectionProps {
  rules: OperationalRule[];
}

export function RulesSection({ rules }: RulesSectionProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <section className="rounded-2xl border border-neutral-800 bg-neutral-900/40 backdrop-blur-sm overflow-hidden transition-all">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex w-full items-center justify-between p-4 text-left hover:bg-neutral-800/30 transition-colors"
      >
        <div className="flex items-center gap-2.5">
          <Compass className="h-4 w-4 text-amber-500" />
          <div>
            <h3 className="text-sm font-semibold text-neutral-200">
              Manual de Manobras & Contingências
            </h3>
            <p className="text-xs text-neutral-500">
              Como agir em caso de atrasos, imprevistos ou fadiga mental
            </p>
          </div>
        </div>
        {isOpen ? (
          <ChevronUp className="h-4 w-4 text-neutral-400" />
        ) : (
          <ChevronDown className="h-4 w-4 text-neutral-400" />
        )}
      </button>

      {isOpen && (
        <div className="border-t border-neutral-800 p-4 grid grid-cols-1 md:grid-cols-2 gap-3">
          {rules.map((rule) => (
            <div
              key={rule.id}
              className="rounded-xl border border-neutral-800/80 bg-neutral-900/80 p-3.5 space-y-1 text-xs"
            >
              <div className="flex items-start gap-1.5 font-medium text-amber-400/90">
                <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                <span>{rule.trigger}</span>
              </div>
              <p className="text-neutral-400 pl-5 leading-relaxed">
                {rule.instruction}
              </p>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
