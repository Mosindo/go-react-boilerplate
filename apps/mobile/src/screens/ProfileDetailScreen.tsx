import React, { useCallback, useState } from "react";
import { FlatList, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { discoverApi } from "../api";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { useFeedback } from "../components/Feedback";
import { Photo } from "../components/Photo";
import { Screen } from "../components/Screen";
import { ActionSheet } from "../components/Sheets";
import { Text } from "../components/Text";
import { useSafetyActions } from "../hooks/useSafetyActions";
import { distanceLabel, genderLabel } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { spacing, useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList, "ProfileDetail">;

export function ProfileDetailScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const { params } = useRoute<RouteProp<RootStackParamList, "ProfileDetail">>();
  const { profile, matchId, canSwipe } = params;
  const { width } = useWindowDimensions();
  const { toast } = useFeedback();
  const { showMatch } = useRealtime();
  const [menuOpen, setMenuOpen] = useState(false);
  const [index, setIndex] = useState(0);
  const [busy, setBusy] = useState(false);
  const pageWidth = Math.min(width, 560);

  const leave = useCallback(() => navigation.popToTop(), [navigation]);
  const safety = useSafetyActions(profile, matchId, leave);

  const swipe = async (action: "like" | "pass") => {
    setBusy(true);
    try {
      const result = await discoverApi.swipe(profile.userId, action);
      navigation.goBack();
      if (result.matched && result.match) showMatch(result.match);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Action impossible.", "error");
    } finally {
      setBusy(false);
    }
  };

  React.useLayoutEffect(() => {
    navigation.setOptions({
      headerRight: () => (
        <Pressable
          onPress={() => setMenuOpen(true)}
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          hitSlop={10}
          testID="profile-menu"
        >
          <Text variant="title">⋯</Text>
        </Pressable>
      )
    });
  }, [navigation]);

  const distance = distanceLabel(profile.distanceKm);

  return (
    <Screen edges={["left", "right"]} scroll padded={false}>
      <View style={{ width: pageWidth, alignSelf: "center" }}>
        <FlatList
          data={profile.photos}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          keyExtractor={(p) => p.id}
          onMomentumScrollEnd={(e) => setIndex(Math.round(e.nativeEvent.contentOffset.x / pageWidth))}
          renderItem={({ item, index: i }) => (
            <Photo
              url={item.url}
              label={`Photo ${i + 1} de ${profile.firstName}`}
              style={{ width: pageWidth, height: pageWidth * 1.25 }}
            />
          )}
        />
        {profile.photos.length > 1 ? (
          <Text variant="caption" muted center style={{ paddingTop: spacing.sm }}>
            {index + 1} / {profile.photos.length} · faites glisser pour voir les autres photos
          </Text>
        ) : null}
      </View>

      <View style={[styles.body, { maxWidth: 560 }]}>
        <Text variant="display" testID="detail-name">
          {profile.firstName}, {profile.age}
        </Text>
        <Text muted>{[genderLabel(profile.gender), profile.jobTitle].filter(Boolean).join(" · ")}</Text>
        <Text muted>{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        {profile.bio ? (
          <View style={styles.section}>
            <Text variant="heading">À propos</Text>
            <Text>{profile.bio}</Text>
          </View>
        ) : null}
        {profile.interests.length ? (
          <View style={styles.section}>
            <Text variant="heading">Centres d&apos;intérêt</Text>
            <View style={styles.chips}>
              {profile.interests.map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
          </View>
        ) : null}
        {canSwipe ? (
          <View style={[styles.actions, { borderTopColor: theme.border }]}>
            <Button
              label="Passer"
              variant="secondary"
              onPress={() => void swipe("pass")}
              disabled={busy}
              style={styles.flex}
            />
            <Button
              label="J'aime"
              onPress={() => void swipe("like")}
              disabled={busy}
              style={styles.flex}
              testID="detail-like"
            />
          </View>
        ) : null}
        {matchId ? (
          <Button
            label={`Écrire à ${profile.firstName}`}
            onPress={() => navigation.navigate("Chat", { matchId, profile })}
          />
        ) : null}
      </View>

      <ActionSheet
        visible={menuOpen}
        title={profile.firstName}
        onClose={() => setMenuOpen(false)}
        actions={[
          ...(matchId
            ? [{ label: "Retirer ce match", onPress: () => void safety.unmatch(), destructive: true }]
            : []),
          { label: "Signaler", onPress: safety.openReport, destructive: true, testID: "action-report" },
          { label: "Bloquer", onPress: () => void safety.block(), destructive: true, testID: "action-block" }
        ]}
      />
      {safety.reportSheet}
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { padding: spacing.lg, gap: spacing.md, width: "100%", alignSelf: "center" },
  section: { gap: spacing.sm, marginTop: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  actions: {
    flexDirection: "row",
    gap: spacing.md,
    paddingTop: spacing.lg,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  flex: { flex: 1 }
});
