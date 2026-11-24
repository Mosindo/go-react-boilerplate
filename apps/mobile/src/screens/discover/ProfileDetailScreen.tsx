import React, { useState } from "react";
import { Alert, Platform, ScrollView, StyleSheet, useWindowDimensions, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Image } from "expo-image";
import { SafeAreaView } from "react-native-safe-area-context";
import Ionicons from "@expo/vector-icons/Ionicons";
import { matchingApi, profileApi, safetyApi } from "../../api/endpoints";
import type { ProfileCard } from "../../api/types";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Chip } from "../../components/Chip";
import { IconButton } from "../../components/IconButton";
import { ReportSheet } from "../../components/safety/ReportSheet";
import { Sheet } from "../../components/Sheet";
import { ErrorState, LoadingState } from "../../components/States";
import { showToast } from "../../components/Toast";
import { removeFromDeck, useSwipe } from "../../hooks/useSwipe";
import type { RootStackParamList } from "../../navigation/types";
import { queryKeys } from "../../queryClient";
import { absoluteUrl } from "../../config";
import { colors, layout, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { distanceLabel, genderLabels, goalLabels } from "../../utils/format";

function confirm(title: string, message: string, confirmLabel: string, onConfirm: () => void) {
  if (Platform.OS === "web") {
    if (globalThis.confirm?.(`${title}\n\n${message}`)) onConfirm();
    return;
  }
  Alert.alert(title, message, [
    { text: "Annuler", style: "cancel" },
    { text: confirmLabel, style: "destructive", onPress: onConfirm }
  ]);
}

export default function ProfileDetailScreen() {
  const navigation = useNavigation();
  const route = useRoute<RouteProp<RootStackParamList, "ProfileDetail">>();
  const { userId, from, matchId } = route.params;
  const client = useQueryClient();
  const swipe = useSwipe();
  const { width } = useWindowDimensions();
  const photoWidth = Math.min(width, layout.maxContentWidth);
  const [page, setPage] = useState(0);
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);

  const cached = client.getQueryData<ProfileCard[]>(queryKeys.discovery)?.find((c) => c.userId === userId);
  const query = useQuery({ queryKey: queryKeys.card(userId), queryFn: () => profileApi.card(userId), initialData: cached });

  const close = () => navigation.goBack();
  const leave = () => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    if (from === "chat") navigation.navigate("Main", { screen: "Matches" });
    else close();
  };

  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) {
    return (
      <SafeAreaView style={styles.safe}>
        <ErrorState message={errorMessage(query.error, "Ce profil n'est plus disponible.")} onRetry={close} />
      </SafeAreaView>
    );
  }
  const card = query.data;
  const distance = distanceLabel(card.distanceKm);

  const decide = (decision: "like" | "pass") => {
    swipe.mutate({ card, decision });
    close();
  };

  const block = () =>
    confirm(`Bloquer ${card.firstName} ?`, "Vous ne vous verrez plus et votre conversation sera supprimée.", "Bloquer", async () => {
      setMenu(false);
      try {
        await safetyApi.block(card.userId);
        removeFromDeck(client, card.userId);
        void client.invalidateQueries({ queryKey: queryKeys.blocks });
        showToast(`${card.firstName} est bloqué·e.`, "success");
        leave();
      } catch (e) {
        showToast(errorMessage(e), "error");
      }
    });

  const unmatch = () =>
    confirm(`Annuler le match avec ${card.firstName} ?`, "La conversation sera supprimée pour vous deux.", "Annuler le match", async () => {
      setMenu(false);
      if (!matchId) return;
      try {
        await matchingApi.unmatch(matchId);
        showToast("Match annulé.", "success");
        leave();
      } catch (e) {
        showToast(errorMessage(e), "error");
      }
    });

  const facts = [
    { icon: "heart-circle-outline" as const, text: goalLabels[card.relationshipGoal] },
    { icon: "person-outline" as const, text: genderLabels[card.gender] },
    card.jobTitle ? { icon: "briefcase-outline" as const, text: card.jobTitle } : null,
    card.heightCm ? { icon: "resize-outline" as const, text: `${card.heightCm} cm` } : null,
    card.city ? { icon: "business-outline" as const, text: card.city } : null,
    distance ? { icon: "location-outline" as const, text: distance } : null
  ].filter((f): f is { icon: "heart-circle-outline"; text: string } => f !== null);

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]} testID="profile-detail">
      <View style={styles.topBar}>
        <IconButton icon="chevron-down" label="Fermer" onPress={close} testID="profile-detail-close" />
        <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} testID="profile-detail-menu" />
      </View>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={[styles.column, { width: photoWidth }]}>
          <ScrollView
            horizontal
            pagingEnabled
            showsHorizontalScrollIndicator={false}
            onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / photoWidth))}
            onScroll={Platform.OS === "web" ? (e) => setPage(Math.round(e.nativeEvent.contentOffset.x / photoWidth)) : undefined}
            scrollEventThrottle={64}
          >
            {card.photos.map((p, i) => (
              <Image
                key={p.id}
                source={{ uri: absoluteUrl(p.url) }}
                style={{ width: photoWidth, height: photoWidth * 1.25 }}
                contentFit="cover"
                accessibilityLabel={`Photo ${i + 1} de ${card.firstName}`}
              />
            ))}
          </ScrollView>
          {card.photos.length > 1 ? (
            <View style={styles.dots}>
              {card.photos.map((p, i) => (
                <View key={p.id} style={[styles.dot, i === page && styles.dotActive]} />
              ))}
            </View>
          ) : null}
          <View style={styles.body}>
            <AppText variant="display">
              {card.firstName}, {card.age}
            </AppText>
            <View style={styles.facts}>
              {facts.map((f) => (
                <View key={f.text} style={styles.fact}>
                  <Ionicons name={f.icon} size={18} color={colors.textMuted} />
                  <AppText color={colors.textMuted}>{f.text}</AppText>
                </View>
              ))}
            </View>
            {card.bio ? (
              <View style={styles.block}>
                <AppText variant="overline" color={colors.textSubtle}>
                  À propos
                </AppText>
                <AppText>{card.bio}</AppText>
              </View>
            ) : null}
            {card.interests.length > 0 ? (
              <View style={styles.block}>
                <AppText variant="overline" color={colors.textSubtle}>
                  Centres d’intérêt
                </AppText>
                <View style={styles.chips}>
                  {card.interests.map((i) => (
                    <Chip key={i.id} label={i.label} highlighted={i.shared} />
                  ))}
                </View>
              </View>
            ) : null}
          </View>
        </View>
      </ScrollView>
      {from === "discover" ? (
        <View style={styles.actions}>
          <IconButton icon="close" label="Passer" size={60} onPress={() => decide("pass")} testID="profile-detail-pass" />
          <IconButton icon="heart" label="J'aime" size={68} tone="primary" onPress={() => decide("like")} testID="profile-detail-like" />
        </View>
      ) : null}

      <Sheet visible={menu} title={card.firstName} onClose={() => setMenu(false)}>
        {from === "chat" && matchId ? <Button label="Annuler le match" icon="heart-dislike-outline" variant="secondary" onPress={unmatch} /> : null}
        <Button label="Bloquer" icon="ban-outline" variant="secondary" onPress={block} testID="profile-block" />
        <Button
          label="Signaler"
          icon="flag-outline"
          variant="danger"
          onPress={() => {
            setMenu(false);
            setReporting(true);
          }}
          testID="profile-report"
        />
      </Sheet>
      <ReportSheet
        visible={reporting}
        userId={card.userId}
        name={card.firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          leave();
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  topBar: { flexDirection: "row", justifyContent: "space-between", paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  content: { alignItems: "center", paddingBottom: spacing.xxl },
  column: { maxWidth: layout.maxContentWidth },
  dots: { flexDirection: "row", justifyContent: "center", gap: 6, marginTop: spacing.sm },
  dot: { width: 6, height: 6, borderRadius: 3, backgroundColor: colors.borderStrong },
  dotActive: { backgroundColor: colors.primary, width: 18 },
  body: { padding: spacing.lg, gap: spacing.lg },
  facts: { gap: spacing.xs },
  fact: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  block: { gap: spacing.xs, padding: spacing.md, borderRadius: radii.md, backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, paddingVertical: spacing.md, borderTopWidth: 1, borderTopColor: colors.border }
});
