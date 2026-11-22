import React, { useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { blockUser, swipe, unmatch } from "../../api/dating";
import { PhotoCarousel } from "../../components/PhotoCarousel";
import { ReportSheet } from "../../components/ReportSheet";
import { qk, useMyProfile, usePublicProfile } from "../../hooks/queries";
import { confirm } from "../../lib/confirm";
import { profileDecided } from "../../lib/events";
import { formatDistance, genderLabels } from "../../lib/format";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Chip, EmptyState, IconButton, LoadingState, Screen, Text, colors, radii, showToast, spacing } from "../../ui";

export default function ProfileDetailScreen({ navigation, route }: MainScreenProps<"ProfileDetail">) {
  const { userId, context } = route.params;
  const client = useQueryClient();
  const profile = usePublicProfile(userId);
  const me = useMyProfile();
  const [busy, setBusy] = useState<"like" | "pass" | "block" | "unmatch" | null>(null);
  const [reporting, setReporting] = useState(false);

  const close = () => (navigation.canGoBack() ? navigation.goBack() : navigation.navigate("Tabs"));

  if (profile.isPending) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen>
        <EmptyState
          actionLabel="Fermer"
          icon="eye-off-outline"
          message="Ce profil n'est plus disponible."
          onAction={close}
          title="Profil introuvable"
        />
      </Screen>
    );
  }
  const p = profile.data;
  const shared = new Set(me.data?.interests.map((i) => i.slug) ?? []);
  const facts = [p.jobTitle, genderLabels[p.gender], p.city, formatDistance(p.distanceKm)].filter(Boolean);

  const decide = async (action: "like" | "pass") => {
    setBusy(action);
    try {
      const result = await swipe(p.userId, action);
      profileDecided.emit(p.userId);
      void client.invalidateQueries({ queryKey: qk.likes });
      if (result.matched && result.match) {
        void client.invalidateQueries({ queryKey: qk.chats });
        showToast(`C'est un match avec ${p.firstName} !`, "success");
        navigation.replace("ChatThread", { userId: p.userId, firstName: p.firstName, photoUrl: p.photos[0]?.url });
        return;
      }
      close();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusy(null);
    }
  };

  const afterRemoval = () => {
    profileDecided.emit(p.userId);
    void client.invalidateQueries({ queryKey: qk.chats });
    void client.invalidateQueries({ queryKey: qk.matches });
    void client.invalidateQueries({ queryKey: qk.likes });
    navigation.navigate("Tabs");
  };

  const block = async () => {
    if (!(await confirm(`Bloquer ${p.firstName} ?`, "Vous ne vous verrez plus et votre conversation sera supprimée.", "Bloquer", true))) {
      return;
    }
    setBusy("block");
    try {
      await blockUser(p.userId);
      showToast(`${p.firstName} est bloqué·e.`, "success");
      afterRemoval();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusy(null);
    }
  };

  const doUnmatch = async () => {
    if (
      !(await confirm(`Annuler le match avec ${p.firstName} ?`, "La conversation sera supprimée pour vous deux.", "Annuler le match", true))
    ) {
      return;
    }
    setBusy("unmatch");
    try {
      await unmatch(p.userId);
      showToast("Match annulé.");
      afterRemoval();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusy(null);
    }
  };

  return (
    <Screen edges={["top", "left", "right", "bottom"]} padded={false} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.photo}>
          <PhotoCarousel name={p.firstName} photos={p.photos} rounded={radii.xl} />
          <IconButton background={colors.overlay} icon="chevron-down" label="Fermer" onPress={close} style={styles.close} />
        </View>
        <View style={styles.body}>
          <Text variant="display">
            {p.firstName}, {p.age}
          </Text>
          {p.isDemo ? (
            <Text tone="accent" variant="caption">
              Profil de démonstration — ce n’est pas un vrai membre.
            </Text>
          ) : null}
          {facts.length ? <Text tone="muted">{facts.join(" · ")}</Text> : null}
          {p.bio ? (
            <View style={styles.block}>
              <Text tone="subtle" variant="eyebrow">
                À propos
              </Text>
              <Text>{p.bio}</Text>
            </View>
          ) : null}
          {p.interests.length ? (
            <View style={styles.block}>
              <Text tone="subtle" variant="eyebrow">
                Centres d’intérêt
              </Text>
              <View style={styles.chips}>
                {p.interests.map((i) => (
                  <Chip highlighted={shared.has(i.slug)} key={i.slug} label={i.label} />
                ))}
              </View>
            </View>
          ) : null}
          <View style={styles.safety}>
            {context === "match" ? (
              <Button
                compact
                icon="heart-dislike-outline"
                label="Annuler le match"
                loading={busy === "unmatch"}
                onPress={doUnmatch}
                variant="secondary"
              />
            ) : null}
            <Button
              compact
              icon="ban-outline"
              label={`Bloquer ${p.firstName}`}
              loading={busy === "block"}
              onPress={block}
              testID="profile-block"
              variant="secondary"
            />
            <Button
              compact
              icon="flag-outline"
              label="Signaler"
              onPress={() => setReporting(true)}
              testID="profile-report"
              variant="danger"
            />
          </View>
        </View>
      </ScrollView>
      {context !== "match" ? (
        <View style={styles.decision}>
          <IconButton
            color={colors.pass}
            disabled={busy !== null}
            icon="close"
            label={`Passer ${p.firstName}`}
            onPress={() => decide("pass")}
            size={60}
            testID="detail-pass"
          />
          <IconButton
            background={colors.likeSoft}
            color={colors.like}
            disabled={busy !== null}
            icon="heart"
            label={`Liker ${p.firstName}`}
            onPress={() => decide("like")}
            size={60}
            testID="detail-like"
          />
        </View>
      ) : (
        <View style={styles.decision}>
          <Button
            icon="chatbubble-ellipses-outline"
            label={`Écrire à ${p.firstName}`}
            onPress={() => navigation.navigate("ChatThread", { userId: p.userId, firstName: p.firstName, photoUrl: p.photos[0]?.url })}
            style={styles.grow}
          />
        </View>
      )}
      <ReportSheet
        firstName={p.firstName}
        onClose={() => setReporting(false)}
        onReported={() => {
          setReporting(false);
          afterRemoval();
        }}
        userId={p.userId}
        visible={reporting}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  scroll: {
    width: "100%",
    maxWidth: 560,
    alignSelf: "center",
    paddingBottom: spacing.xxl
  },
  photo: {
    aspectRatio: 4 / 5,
    margin: spacing.sm
  },
  close: {
    position: "absolute",
    top: spacing.md,
    right: spacing.md
  },
  body: {
    paddingHorizontal: spacing.lg,
    gap: spacing.sm
  },
  block: {
    gap: spacing.xs,
    marginTop: spacing.md
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  },
  safety: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs,
    marginTop: spacing.xl
  },
  decision: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  },
  grow: {
    flex: 1,
    maxWidth: 420
  }
});
