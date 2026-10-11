import { useState } from 'react';
import { fr } from '../../i18n';
import { useRootNavigation } from '../../navigation/useRootNavigation';
import { ActionSheet, ConfirmDialog } from '../../ui';
import type { SheetAction } from '../../ui';
import { useBlockUser, useUnmatch } from './hooks';
import type { SafetyTarget } from './hooks';

interface UserActionsSheetProps {
  visible: boolean;
  onClose: () => void;
  target: SafetyTarget;
  /** Extra leading actions (e.g. "View profile"). */
  leading?: SheetAction[];
  /** Called after block / unmatch succeeded (typically navigates back). */
  onGone: () => void;
}

/** Shared "…" menu for a person: report, block and (when matched) unmatch, each with confirmation. */
export function UserActionsSheet({
  visible,
  onClose,
  target,
  leading = [],
  onGone,
}: UserActionsSheetProps) {
  const navigation = useRootNavigation();
  const [confirm, setConfirm] = useState<'block' | 'unmatch' | null>(null);
  const block = useBlockUser(() => {
    setConfirm(null);
    onGone();
  });
  const unmatch = useUnmatch(() => {
    setConfirm(null);
    onGone();
  });

  const actions: SheetAction[] = [
    ...leading,
    ...(target.matchId ? [{ label: fr.safety.unmatch, onPress: () => setConfirm('unmatch') }] : []),
    {
      label: fr.safety.report,
      onPress: () =>
        navigation.navigate('Report', { userId: target.userId, firstName: target.firstName }),
    },
    { label: fr.safety.block, variant: 'danger' as const, onPress: () => setConfirm('block') },
  ];

  return (
    <>
      <ActionSheet visible={visible} onClose={onClose} title={target.firstName} actions={actions} />
      <ConfirmDialog
        visible={confirm === 'block'}
        title={fr.safety.blockTitle(target.firstName)}
        message={fr.safety.blockMessage}
        confirmLabel={fr.safety.block}
        destructive
        loading={block.isPending}
        onConfirm={() => block.mutate(target)}
        onCancel={() => setConfirm(null)}
      />
      <ConfirmDialog
        visible={confirm === 'unmatch'}
        title={fr.safety.unmatchTitle(target.firstName)}
        message={fr.safety.unmatchMessage}
        confirmLabel={fr.safety.unmatch}
        destructive
        loading={unmatch.isPending}
        onConfirm={() => unmatch.mutate(target)}
        onCancel={() => setConfirm(null)}
      />
    </>
  );
}
