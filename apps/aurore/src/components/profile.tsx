import React, { useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Card as ProfileCard } from "../api/types";
import { distanceLabel, genderLabel, headline } from "../lib/format";
import { radii, spacing, useTheme } from "../theme/theme";
import { AuthImage } from "./AuthImage";
import { Button, Chip, Row, Text } from "./ui";

/** Photo area with tap-left / tap-right navigation and progress dashes. */
export function PhotoCarousel({ card, height, fill, onTap }: { card: ProfileCard; height?: number; fill?: boolean; onTap?: () => void }) {
  const t = useTheme();
  const [index, setIndex] = useState(0);
  const count = card.photos.length;
  const current = card.photos[Math.min(index, Math.max(count - 1, 0))];
  return (
    <View style={[styles.carousel, fill ? StyleSheet.absoluteFillObject : height ? { height } : { aspectRatio: 3 / 4 }, { backgroundColor: t.surfaceAlt }]}>
      <AuthImage path={current?.url} label={`Photo de ${card.firstName}`} style={StyleSheet.absoluteFill as object} />
      <View style={styles.dashes} pointerEvents="none">
        {card.photos.map((p, i) => (
          <View key={p.id} style={[styles.dash, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
        ))}
      </View>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Photo précédente"
        style={[styles.tapZone, { left: 0 }]}
        onPress={() => (index > 0 ? setIndex(index - 1) : onTap?.())}
      />
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Photo suivante"
        style={[styles.tapZone, { right: 0 }]}
        onPress={() => (index < count - 1 ? setIndex(index + 1) : onTap?.())}
      />
    </View>
  );
}

export function CardOverlay({ card }: { card: ProfileCard }) {
  const km = distanceLabel(card.distanceKm);
  return (
    <View style={styles.overlay} pointerEvents="none">
      <Text variant="title" color="#FFFFFF" style={styles.shadow}>
        {headline(card)}
      </Text>
      {km || card.city ? (
        <View style={{ flexDirection: "row", alignItems: "center", marginTop: 2 }}>
          <Ionicons name="location" size={15} color="#FFFFFF" />
          <Text variant="label" color="#FFFFFF" style={[styles.shadow, { marginLeft: 4 }]}>
            {[card.city, km].filter(Boolean).join(" · ")}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

export function ProfileBody({ card }: { card: ProfileCard }) {
  const t = useTheme();
  const km = distanceLabel(card.distanceKm);
  return (
    <View style={{ padding: spacing.lg }}>
      <Text variant="title">{headline(card)}</Text>
      <Text muted style={{ marginTop: 2 }}>
        {[genderLabel(card.gender), card.city, km].filter(Boolean).join(" · ")}
      </Text>
      {card.bio ? (
        <View style={{ marginTop: spacing.lg }}>
          <Text variant="label" muted style={{ marginBottom: spacing.xs }}>
            À propos
          </Text>
          <Text>{card.bio}</Text>
        </View>
      ) : null}
      {card.interests.length > 0 ? (
        <View style={{ marginTop: spacing.lg }}>
          <Text variant="label" muted style={{ marginBottom: spacing.sm }}>
            Centres d&apos;intérêt
          </Text>
          <Row>
            {card.interests.map((i) => (
              <Chip key={i.id} label={i.label} />
            ))}
          </Row>
        </View>
      ) : null}
      {card.photos.length > 1 ? (
        <View style={{ marginTop: spacing.lg, gap: spacing.md }}>
          {card.photos.slice(1).map((p) => (
            <View key={p.id} style={{ borderRadius: radii.lg, overflow: "hidden", aspectRatio: 3 / 4, backgroundColor: t.surfaceAlt }}>
              <AuthImage path={p.url} label={`Photo de ${card.firstName}`} style={StyleSheet.absoluteFill as object} />
            </View>
          ))}
        </View>
      ) : null}
    </View>
  );
}

export function MatchModal({
  match,
  onMessage,
  onClose
}: {
  match: ProfileCard | null;
  onMessage: () => void;
  onClose: () => void;
}) {
  const t = useTheme();
  return (
    <Modal transparent visible={!!match} animationType="fade" onRequestClose={onClose}>
      <View style={[styles.matchBackdrop, { backgroundColor: t.overlay }]}>
        <View style={[styles.matchCard, { backgroundColor: t.surface }]} testID="match-modal">
          <Ionicons name="heart" size={44} color={t.primary} />
          <Text variant="display" style={{ textAlign: "center", marginTop: spacing.sm }}>
            C&apos;est un match !
          </Text>
          <Text muted style={{ textAlign: "center", marginVertical: spacing.lg }}>
            Vous et {match?.firstName} vous plaisez mutuellement. Dites bonjour !
          </Text>
          <Button label="Envoyer un message" icon="chatbubble" onPress={onMessage} testID="match-message" />
          <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} style={{ marginTop: spacing.sm }} testID="match-continue" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  carousel: { width: "100%", overflow: "hidden" },
  dashes: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  dash: { flex: 1, height: 3, borderRadius: 2 },
  tapZone: { position: "absolute", top: 0, bottom: 0, width: "40%" },
  overlay: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    backgroundColor: "rgba(20,10,22,0.45)"
  },
  shadow: { textShadowColor: "rgba(0,0,0,0.5)", textShadowRadius: 6, textShadowOffset: { width: 0, height: 1 } },
  matchBackdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  matchCard: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "stretch" }
});
