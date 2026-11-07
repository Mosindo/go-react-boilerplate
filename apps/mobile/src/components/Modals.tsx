import React, { useState } from "react";
import { Modal, Pressable, View } from "react-native";
import { AppText, Avatar, Banner, Button, Chip, Row, TextField } from "./ui";
import { radius, spacing, useTheme } from "../theme/theme";
import { safetyApi } from "../api/services";
import { errorMessage } from "../api/http";
import type { MatchView, ReportReason } from "../api/types";

export function Sheet({
  visible,
  onClose,
  children,
}: {
  visible: boolean;
  onClose: () => void;
  children: React.ReactNode;
}) {
  const { colors } = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        style={{ flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" }}
        onPress={onClose}
        accessibilityLabel="Fermer"
      >
        <Pressable
          onPress={() => undefined}
          style={{
            backgroundColor: colors.bg,
            borderTopLeftRadius: radius.lg,
            borderTopRightRadius: radius.lg,
            padding: spacing.xl,
            gap: spacing.lg,
          }}
        >
          {children}
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "harassment", label: "Harcèlement" },
  { value: "fake", label: "Faux profil" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Mineur" },
  { value: "other", label: "Autre" },
];

export function ReportSheet({
  visible,
  userId,
  name,
  onClose,
  onDone,
}: {
  visible: boolean;
  userId: string;
  name: string;
  onClose: () => void;
  onDone: (blocked: boolean) => void;
}) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) return setError("Choisissez un motif.");
    setBusy(true);
    setError(null);
    try {
      await safetyApi.report(userId, reason, details.trim(), alsoBlock);
      setReason(null);
      setDetails("");
      onDone(alsoBlock);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet visible={visible} onClose={onClose}>
      <AppText variant="heading">Signaler {name}</AppText>
      <AppText variant="muted">Votre signalement est confidentiel. Notre équipe l&apos;examinera.</AppText>
      {error ? <Banner message={error} /> : null}
      <Row style={{ flexWrap: "wrap", gap: spacing.sm }}>
        {REASONS.map((r) => (
          <Chip
            key={r.value}
            testID={`reason-${r.value}`}
            label={r.label}
            selected={reason === r.value}
            onPress={() => setReason(r.value)}
          />
        ))}
      </Row>
      <TextField
        label="Détails (facultatif)"
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
      />
      <Chip
        label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
        selected={alsoBlock}
        onPress={() => setAlsoBlock((v) => !v)}
      />
      <Button
        testID="report-submit"
        title="Envoyer le signalement"
        variant="danger"
        loading={busy}
        onPress={submit}
      />
      <Button title="Annuler" variant="ghost" onPress={onClose} />
    </Sheet>
  );
}

export function MatchModal({
  match,
  onChat,
  onClose,
}: {
  match: MatchView | null;
  onChat: () => void;
  onClose: () => void;
}) {
  const { colors } = useTheme();
  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose}>
      <View
        style={{
          flex: 1,
          backgroundColor: colors.overlay,
          alignItems: "center",
          justifyContent: "center",
          padding: spacing.xl,
        }}
      >
        <View
          testID="match-modal"
          style={{
            backgroundColor: colors.bg,
            borderRadius: radius.lg,
            padding: spacing.xl,
            gap: spacing.lg,
            alignItems: "center",
            width: "100%",
            maxWidth: 380,
          }}
        >
          <AppText variant="title" style={{ color: colors.primary }}>
            C&apos;est un match !
          </AppText>
          {match ? <Avatar name={match.user.firstName} uri={match.user.photos[0]?.url} size={112} /> : null}
          <AppText style={{ textAlign: "center" }}>
            Vous et {match?.user.firstName} vous plaisez mutuellement. Dites quelque chose de sympa !
          </AppText>
          <Button
            testID="match-chat"
            title="Envoyer un message"
            onPress={onChat}
            style={{ alignSelf: "stretch" }}
          />
          <Button testID="match-continue" title="Continuer à découvrir" variant="ghost" onPress={onClose} />
        </View>
      </View>
    </Modal>
  );
}
