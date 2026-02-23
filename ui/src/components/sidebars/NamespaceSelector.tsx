"use client";

import { useNamespace } from "@/contexts/NamespaceContext";
import { NamespaceCombobox } from "@/components/NamespaceCombobox";

export function NamespaceSelector() {
  const { namespace, setNamespace } = useNamespace();
  return (
    <div className="px-1 py-1 group-data-[collapsible=icon]:hidden">
      <NamespaceCombobox
        value={namespace}
        onValueChange={setNamespace}
        placeholder="Select namespace..."
      />
    </div>
  );
}
