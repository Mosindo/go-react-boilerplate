import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { keys } from "../../hooks/queries";
import { spacing } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { ActionSheet, ConfirmDialog, Sheet, type ActionItem } from "../../ui/Modal";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";
import { useToast } from "../../ui/Toast";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Mineur(e)" },
  { value: "other", label: "Autre" }
];

type Options = { userId: string; firstName: string; extraActions?: ActionItem[]; onBlocked?: () => void };

/** Report / block menu shared by the profile and chat screens. Returns `open` and the elements to render. */
export function useSafetyMenu({ userId, firstName, extraActions = [], onBlocked }: Options) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const [blocking, setBlocking] = useState(false);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const refreshAll = () => {
    void queryClient.invalidateQueries({ queryKey: keys.discover });
    void queryClient.invalidateQueries({ queryKey: keys.matches });
    void queryClient.invalidateQueries({ queryKey: keys.conversations });
    void queryClient.invalidateQueries({ queryKey: keys.blocked });
    void queryClient.invalidateQueries({ queryKey: keys.notifications });
  };
  const fail = (e: unknown) => toast.show(e instanceof ApiError ? e.message : "Action impossible.", "error");

  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      setReporting(false);
      setReason(null);
      setDetails("");
      toast.show("Merci, ton signalement a bien été envoyé.", "success");
    },
    onError: fail
  });

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      setBlocking(false);
      refreshAll();
      toast.show(`${firstName} est bloqué(e).`, "success");
      onBlocked?.();
    },
    onError: fail
  });

  const element = (
    <>
      <ActionSheet
        visible={menu}
        onClose={() => setMenu(false)}
        actions={[
          ...extraActions,
          { label: `Signaler ${firstName}`, onPress: () => setReporting(true) },
          { label: `Bloquer ${firstName}`, destructive: true, onPress: () => setBlocking(true) }
        ]}
      />
      <Sheet visible={reporting} onClose={() => setReporting(false)} title={`Signaler ${firstName}`}>
        <Text tone="textMuted" style={styles.gap}>
          Pourquoi signales-tu ce profil ? Notre équipe examinera ton signalement.
        </Text>
        <View style={styles.chips}>
          {REASONS.map((r) => (
            <Chip
              key={r.value}
              label={r.label}
              selected={reason === r.value}
              onPress={() => setReason(r.value)}
            />
          ))}
        </View>
        <TextField
          label="Détails (facultatif)"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
        />
        <Button
          title="Envoyer le signalement"
          onPress={() => report.mutate()}
          disabled={!reason}
          loading={report.isPending}
        />
        <Button title="Annuler" variant="ghost" onPress={() => setReporting(false)} />
      </Sheet>
      <ConfirmDialog
        visible={blocking}
        title={`Bloquer ${firstName} ?`}
        message="Vous ne pourrez plus vous voir ni vous écrire. Tu pourras le/la débloquer depuis Confidentialité."
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        onConfirm={() => block.mutate()}
        onCancel={() => setBlocking(false)}
      />
    </>
  );

  return { open: () => setMenu(true), element };
}

const styles = StyleSheet.create({
  gap: { marginBottom: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", marginBottom: spacing.sm }
});
