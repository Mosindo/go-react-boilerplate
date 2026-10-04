import React, { useState, type ReactNode } from "react";
import { getErrorMessage } from "../../api/client";
import { useBlockUser } from "../../hooks/useSafety";
import { ActionSheet, ConfirmDialog, toast, type SheetAction } from "../../ui";
import { ReportModal } from "./ReportModal";

type Target = { userId: string; displayName: string };

type Options = {
  target: Target;
  /** Extra actions shown above report/block (e.g. "View profile", "Unmatch"). */
  extraActions?: SheetAction[];
  /** Called after the person was blocked (directly or through a report). */
  onBlocked?: () => void;
};

/** Menu + confirm + report flow for another user. Render `element` once; call `open()` to show it. */
export function useSafetyMenu({ target, extraActions = [], onBlocked }: Options): {
  open: () => void;
  element: ReactNode;
} {
  const [menu, setMenu] = useState(false);
  const [confirmBlock, setConfirmBlock] = useState(false);
  const [report, setReport] = useState(false);
  const block = useBlockUser();

  async function doBlock() {
    try {
      await block.mutateAsync(target.userId);
      setConfirmBlock(false);
      toast.success(`${target.displayName} is blocked.`);
      onBlocked?.();
    } catch (e) {
      setConfirmBlock(false);
      toast.error(getErrorMessage(e));
    }
  }

  const element = (
    <>
      <ActionSheet
        actions={[
          ...extraActions,
          { label: "Report", icon: "flag-outline", onPress: () => setReport(true) },
          { label: "Block", icon: "ban-outline", destructive: true, onPress: () => setConfirmBlock(true) }
        ]}
        onClose={() => setMenu(false)}
        title={target.displayName}
        visible={menu}
      />
      <ConfirmDialog
        confirmLabel="Block"
        destructive
        loading={block.isPending}
        message="They won't be able to see your profile or message you, and any match with them will be removed. You can unblock them later from your profile."
        onCancel={() => setConfirmBlock(false)}
        onConfirm={doBlock}
        title={`Block ${target.displayName}?`}
        visible={confirmBlock}
      />
      <ReportModal
        displayName={target.displayName}
        onClose={() => setReport(false)}
        onDone={(blocked) => {
          if (blocked) onBlocked?.();
        }}
        userId={target.userId}
        visible={report}
      />
    </>
  );

  return { open: () => setMenu(true), element };
}
