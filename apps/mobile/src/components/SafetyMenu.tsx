import React, { useState } from "react";
import { Alert, Platform, TextInput, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../api/client";
import { chatApi, safetyApi } from "../api/endpoints";
import { REPORT_REASONS, type ReportReason } from "../api/types";
import { spacing, useTheme } from "../theme/theme";
import { useToast } from "./Toast";
import { ActionSheet, AppText, Button, Chip, Sheet, type SheetAction } from "./ui";

type Props = {
  visible: boolean;
  onClose: () => void;
  userId: string;
  name: string;
  matchId?: string;
  /** Called once the user is gone from the viewer's world (blocked / unmatched). */
  onGone: () => void;
  extraActions?: SheetAction[];
};

function confirm(title: string, message: string, confirmLabel: string, onConfirm: () => void) {
  if (Platform.OS === "web") {
    onConfirm();
    return;
  }
  Alert.alert(title, message, [
    { text: "Annuler", style: "cancel" },
    { text: confirmLabel, style: "destructive", onPress: onConfirm }
  ]);
}

/** Block / report / unmatch actions shared by the profile and chat screens. */
export function SafetyMenu({ visible, onClose, userId, name, matchId, onGone, extraActions = [] }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [reporting, setReporting] = useState(false);

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    void queryClient.invalidateQueries({ queryKey: ["blocked"] });
  };

  const block = () =>
    confirm(`Bloquer ${name} ?`, "Vous ne vous verrez plus et la conversation sera supprimée.", "Bloquer", async () => {
      try {
        await safetyApi.block(userId);
        invalidate();
        toast.show(`${name} est bloqué·e.`, "success");
        onGone();
      } catch (e) {
        toast.show(apiErrorMessage(e), "error");
      }
    });

  const unmatch = () =>
    confirm(`Supprimer le match avec ${name} ?`, "La conversation sera définitivement supprimée pour vous deux.", "Supprimer", async () => {
      if (!matchId) return;
      try {
        await chatApi.unmatch(matchId);
        invalidate();
        toast.show("Match supprimé.", "success");
        onGone();
      } catch (e) {
        toast.show(apiErrorMessage(e), "error");
      }
    });

  const actions: SheetAction[] = [
    ...extraActions,
    ...(matchId ? [{ label: "Supprimer le match", destructive: true, onPress: unmatch }] : []),
    { label: "Signaler", onPress: () => setReporting(true) },
    { label: `Bloquer ${name}`, destructive: true, onPress: block }
  ];

  return (
    <>
      <ActionSheet visible={visible} onClose={onClose} title={name} actions={actions} />
      <ReportSheet
        visible={reporting}
        onClose={() => setReporting(false)}
        userId={userId}
        name={name}
        onReported={() => {
          setReporting(false);
          toast.show("Merci, votre signalement a été transmis.", "success");
        }}
      />
    </>
  );
}

function ReportSheet({
  visible,
  onClose,
  userId,
  name,
  onReported
}: {
  visible: boolean;
  onClose: () => void;
  userId: string;
  name: string;
  onReported: () => void;
}) {
  const { colors } = useTheme();
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) return;
    setBusy(true);
    try {
      await safetyApi.report(userId, reason, details.trim());
      setReason(null);
      setDetails("");
      onReported();
    } catch (e) {
      toast.show(apiErrorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet visible={visible} onClose={onClose} title={`Signaler ${name}`}>
      <View style={{ padding: spacing.lg, gap: spacing.md }}>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {REPORT_REASONS.map((r) => (
            <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} />
          ))}
        </View>
        <TextInput
          accessibilityLabel="Détails du signalement"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
          placeholder="Détails (facultatif)"
          placeholderTextColor={colors.textMuted}
          style={{
            minHeight: 80,
            borderWidth: 1.5,
            borderColor: colors.border,
            borderRadius: 12,
            padding: spacing.md,
            color: colors.text,
            textAlignVertical: "top"
          }}
        />
        <AppText variant="caption" muted>
          Votre signalement est confidentiel. Pensez aussi à bloquer la personne.
        </AppText>
        <Button label="Envoyer le signalement" onPress={submit} disabled={!reason} loading={busy} />
      </View>
    </Sheet>
  );
}
