import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { reportUser } from "../api/dating";
import type { ReportReason } from "../api/types";
import { Button, Text, TextField, colors, radii, showToast, spacing } from "../ui";
import { ChoiceList } from "./ChoiceList";

const reasons: { value: ReportReason; label: string; description?: string }[] = [
  { value: "fake_profile", label: "Faux profil", description: "Photos volées, identité usurpée" },
  { value: "inappropriate_content", label: "Contenu inapproprié", description: "Photos ou propos choquants" },
  { value: "harassment", label: "Harcèlement ou menaces" },
  { value: "spam", label: "Spam ou arnaque", description: "Liens, demandes d'argent" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type Props = {
  userId: string;
  firstName: string;
  visible: boolean;
  onClose: () => void;
  onReported: () => void;
};

/** Reporting always blocks the member as well, so the reporter is protected immediately. */
export function ReportSheet({ userId, firstName, visible, onClose, onReported }: Props) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [sending, setSending] = useState(false);

  const submit = async () => {
    if (!reason) {
      showToast("Choisissez un motif.", "error");
      return;
    }
    setSending(true);
    try {
      await reportUser(userId, reason, details.trim());
      showToast(`Merci. ${firstName} a été signalé·e et bloqué·e.`, "success");
      setReason(null);
      setDetails("");
      onReported();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setSending(false);
    }
  };

  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={styles.backdrop} />
      <View accessibilityViewIsModal style={styles.sheet} testID="report-sheet">
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <Text variant="heading">Signaler {firstName}</Text>
          <Text tone="muted" variant="caption">
            Votre signalement est confidentiel. {firstName} sera aussi bloqué·e : vous ne vous verrez plus.
          </Text>
          <ChoiceList onToggle={setReason} options={reasons} selected={reason ? [reason] : []} testIDPrefix="report-reason" />
          <TextField
            label="Détails (facultatif)"
            maxLength={1000}
            multiline
            onChangeText={setDetails}
            placeholder="Ce qui s'est passé…"
            value={details}
          />
          <Button label="Envoyer le signalement" loading={sending} onPress={submit} testID="report-submit" variant="danger" />
          <Button label="Annuler" onPress={onClose} variant="ghost" />
        </ScrollView>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay
  },
  sheet: {
    maxHeight: "88%",
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    backgroundColor: colors.surface
  },
  content: {
    gap: spacing.md,
    padding: spacing.lg,
    paddingBottom: spacing.xxl,
    width: "100%",
    maxWidth: 560,
    alignSelf: "center"
  }
});
