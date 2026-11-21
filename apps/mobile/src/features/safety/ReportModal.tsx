import React, { useState } from "react";
import { Modal, Pressable, ScrollView, View } from "react-native";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { makeStyles } from "../../theme";
import { Button, Icon, modalAnimation, showToast, Text, TextField } from "../../ui";

const reasons: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil ou usurpation" },
  { value: "inappropriate_content", label: "Photos ou contenu inapproprié" },
  { value: "harassment", label: "Harcèlement ou propos offensants" },
  { value: "spam", label: "Spam, arnaque ou publicité" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre raison" }
];

type Props = { userId: string; firstName: string; visible: boolean; onClose: () => void; onReported: () => void };

export function ReportModal({ userId, firstName, visible, onClose, onReported }: Props) {
  const styles = useStyles();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!reason) return;
    setLoading(true);
    try {
      await safetyApi.report(userId, reason, details);
      showToast(`Merci. ${firstName} est bloqué·e et notre équipe va examiner le signalement.`, "success");
      onClose();
      onReported();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal visible={visible} animationType={modalAnimation("slide")} transparent onRequestClose={onClose}>
      <View style={styles.backdrop}>
        <View style={styles.sheet} accessibilityViewIsModal>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="title" accessibilityRole="header">
              Signaler {firstName}
            </Text>
            <Text tone="muted">Votre signalement est confidentiel. Ce profil sera aussi bloqué : vous ne vous verrez plus.</Text>
            <View accessibilityRole="radiogroup" style={styles.reasons}>
              {reasons.map((r) => (
                <Pressable
                  key={r.value}
                  testID={`report-${r.value}`}
                  accessibilityRole="radio"
                  accessibilityState={{ checked: reason === r.value }}
                  onPress={() => setReason(r.value)}
                  style={[styles.reason, reason === r.value ? styles.reasonSelected : null]}
                >
                  <Icon
                    name={reason === r.value ? "radio-button-on" : "radio-button-off"}
                    size={20}
                    color={reason === r.value ? "primary" : "textSubtle"}
                  />
                  <Text variant="bodyStrong">{r.label}</Text>
                </Pressable>
              ))}
            </View>
            <TextField
              label="Détails (facultatif)"
              value={details}
              onChangeText={setDetails}
              multiline
              counter={1000}
              placeholder="Ce qui s'est passé…"
            />
            <Button
              testID="report-submit"
              label="Envoyer le signalement"
              variant="primary"
              size="lg"
              disabled={!reason}
              loading={loading}
              onPress={submit}
            />
            <Button label="Annuler" variant="ghost" onPress={onClose} />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const useStyles = makeStyles((t) => ({
  backdrop: { flex: 1, backgroundColor: t.colors.overlay, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    maxHeight: "92%",
    backgroundColor: t.colors.background,
    borderTopLeftRadius: t.radii.xl,
    borderTopRightRadius: t.radii.xl
  },
  content: { padding: t.spacing.xl, gap: t.spacing.lg },
  reasons: { gap: t.spacing.sm },
  reason: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.md,
    padding: t.spacing.md,
    borderRadius: t.radii.md,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface
  },
  reasonSelected: { borderColor: t.colors.primary, backgroundColor: t.colors.primarySoft }
}));
