import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, reportUser } from "../../api/platform";
import { errorMessage } from "../../api/client";
import type { ReportReason } from "../../api/types";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../shared/feedback";
import { ActionSheet, Chip, ConfirmDialog, Input, Notice, Text, Toggle, spacing } from "../../shared/ui";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type SafetyMenuProps = {
  visible: boolean;
  userId: string;
  name: string;
  onClose: () => void;
  /** Called after the user was blocked (or reported and blocked). */
  onBlocked: () => void;
};

/** Block / report entry point shared by the deck, profile pages and chats. */
export function SafetyMenu({ visible, userId, name, onClose, onBlocked }: SafetyMenuProps) {
  const queryClient = useQueryClient();
  const [dialog, setDialog] = useState<"block" | "report" | null>(null);
  const [reason, setReason] = useState<ReportReason>("fake");
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const afterBlock = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.matches }),
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations }),
      queryClient.invalidateQueries({ queryKey: queryKeys.blocked })
    ]);
    setDialog(null);
    onBlocked();
  };

  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: async () => {
      showToast(`${name} est bloqué(e).`);
      await afterBlock();
    },
    onError: (e) => setError(errorMessage(e))
  });

  const report = useMutation({
    mutationFn: () => reportUser(userId, reason, details.trim(), alsoBlock),
    onSuccess: async () => {
      showToast("Merci, votre signalement a été transmis.");
      if (alsoBlock) {
        await afterBlock();
      } else {
        setDialog(null);
      }
    },
    onError: (e) => setError(errorMessage(e))
  });

  return (
    <>
      <ActionSheet
        actions={[
          {
            label: `Signaler ${name}`,
            onPress: () => {
              setError(null);
              setDialog("report");
            }
          },
          {
            label: `Bloquer ${name}`,
            destructive: true,
            onPress: () => {
              setError(null);
              setDialog("block");
            }
          }
        ]}
        onClose={onClose}
        title="Sécurité"
        visible={visible}
      />
      <ConfirmDialog
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        message={`${name} ne pourra plus vous voir ni vous écrire, et vous ne verrez plus ce profil. Vous pourrez débloquer depuis les réglages.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => block.mutate()}
        title={`Bloquer ${name} ?`}
        visible={dialog === "block"}
      >
        {error ? <Notice message={error} tone="danger" /> : null}
      </ConfirmDialog>
      <ConfirmDialog
        confirmLabel="Envoyer le signalement"
        loading={report.isPending}
        onCancel={() => setDialog(null)}
        onConfirm={() => report.mutate()}
        title={`Signaler ${name}`}
        visible={dialog === "report"}
      >
        <View style={styles.reasons}>
          {REASONS.map((r) => (
            <Chip key={r.value} label={r.label} onPress={() => setReason(r.value)} selected={reason === r.value} />
          ))}
        </View>
        <Input label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
        <Toggle label="Bloquer aussi cette personne" onChange={setAlsoBlock} value={alsoBlock} />
        <Text tone="subtle" variant="caption">
          Notre équipe examine chaque signalement. La personne n'est pas prévenue.
        </Text>
        {error ? <Notice message={error} tone="danger" /> : null}
      </ConfirmDialog>
    </>
  );
}

const styles = StyleSheet.create({ reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs } });
