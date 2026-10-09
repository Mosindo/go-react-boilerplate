import React, { useState } from 'react';
import { StyleSheet } from 'react-native';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { safetyApi } from '../api/safety';
import { REPORT_REASONS, type ReportReason } from '../api/types';
import { fr } from '../i18n/fr';
import { qk } from '../query/keys';
import { bumpDiscoverEpoch } from '../store/appState';
import { spacing } from '../theme';
import { toast } from './toastStore';
import {
  ActionSheet,
  Button,
  ChoiceGroup,
  ConfirmDialog,
  Sheet,
  TextField,
  type ActionItem,
} from './ui';

const reasonChoices = REPORT_REASONS.map((r) => ({ value: r, label: fr.safety.reasons[r] }));

/** Report / block flows shared by the profile screen, the chat header and the discover card. */
export function UserActionsMenu({
  visible,
  onClose,
  user,
  extraItems = [],
  onBlocked,
}: {
  visible: boolean;
  onClose: () => void;
  user: { id: string; firstName: string };
  extraItems?: ActionItem[];
  onBlocked?: () => void;
}) {
  const qc = useQueryClient();
  const [dialog, setDialog] = useState<'report' | 'block' | null>(null);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState('');

  const report = useMutation({
    mutationFn: () => safetyApi.report(user.id, reason!, details),
    onSuccess: () => {
      setDialog(null);
      setReason(null);
      setDetails('');
      toast.success(fr.safety.reported);
    },
  });

  const block = useMutation({
    mutationFn: () => safetyApi.block(user.id),
    onSuccess: () => {
      setDialog(null);
      toast.success(fr.safety.blocked(user.firstName));
      void qc.invalidateQueries({ queryKey: qk.conversations });
      void qc.invalidateQueries({ queryKey: qk.matches });
      void qc.invalidateQueries({ queryKey: qk.likes });
      void qc.invalidateQueries({ queryKey: qk.blocks });
      qc.removeQueries({ queryKey: qk.profile(user.id) });
      bumpDiscoverEpoch();
      onBlocked?.();
    },
  });

  const items: ActionItem[] = [
    ...extraItems,
    {
      key: 'report',
      label: fr.safety.report,
      icon: 'flag-outline',
      onPress: () => setDialog('report'),
    },
    {
      key: 'block',
      label: fr.safety.block,
      icon: 'ban-outline',
      destructive: true,
      onPress: () => setDialog('block'),
    },
  ];

  return (
    <>
      <ActionSheet
        visible={visible}
        onClose={onClose}
        items={items}
        title={fr.safety.menuTitle}
        testID="user-menu"
      />
      <Sheet
        visible={dialog === 'report'}
        onClose={() => setDialog(null)}
        title={fr.safety.reportTitle(user.firstName)}
        testID="report-sheet"
      >
        <ChoiceGroup
          mode="single"
          choices={reasonChoices}
          selected={reason ? [reason] : []}
          onToggle={setReason}
          testIDPrefix="report-reason"
          label={fr.safety.reportSubtitle}
        />
        <TextField
          testID="report-details"
          label={fr.safety.details}
          placeholder={fr.safety.detailsPlaceholder}
          value={details}
          onChangeText={setDetails}
          multiline
          counter={1000}
          maxLength={1000}
        />
        <Button
          label={fr.safety.send}
          onPress={() => report.mutate()}
          disabled={!reason}
          loading={report.isPending}
          testID="report-submit"
        />
        <Button
          label={fr.common.cancel}
          variant="ghost"
          onPress={() => setDialog(null)}
          style={styles.gap}
        />
      </Sheet>
      <ConfirmDialog
        visible={dialog === 'block'}
        title={fr.safety.blockTitle(user.firstName)}
        body={fr.safety.blockBody}
        confirmLabel={fr.safety.block}
        destructive
        loading={block.isPending}
        onCancel={() => setDialog(null)}
        onConfirm={() => block.mutate()}
      />
    </>
  );
}

const styles = StyleSheet.create({ gap: { marginTop: spacing.sm } });
