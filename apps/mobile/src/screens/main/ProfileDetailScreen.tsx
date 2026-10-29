import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import React, { useEffect, useState } from "react";
import {
  FlatList,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
  useWindowDimensions,
  type NativeScrollEvent,
  type NativeSyntheticEvent
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { swipe } from "../../api/discovery";
import { fetchPublicProfile } from "../../api/profile";
import type { SwipeAction } from "../../api/types";
import { AuthImage } from "../../components/AuthImage";
import { Chip } from "../../components/Chip";
import { SafetySheet } from "../../components/SafetySheet";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { queryKeys } from "../../hooks/queries";
import { errorMessage } from "../../hooks/useAsyncAction";
import { formatDistance } from "../../lib/format";
import type { RootStackParamList } from "../../navigation/types";
import { useMe } from "../../state/AuthProvider";
import { markProfileResolved } from "../../state/discoverStore";
import { useRealtime } from "../../state/RealtimeProvider";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, firstName, allowActions } = route.params;
  const theme = useTheme();
  const toast = useToast();
  const me = useMe();
  const { announceMatch } = useRealtime();
  const { width } = useWindowDimensions();
  const [page, setPage] = useState(0);
  const [safetyOpen, setSafetyOpen] = useState(false);
  const [acting, setActing] = useState(false);
  const isSelf = userId === me.id;

  const query = useQuery({
    queryKey: queryKeys.publicProfile(userId),
    queryFn: () => fetchPublicProfile(userId)
  });
  const profile = query.data;

  useEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? firstName ?? "Profil",
      headerRight: isSelf
        ? undefined
        : () => (
            <Pressable
              accessibilityLabel="Signaler ou bloquer"
              accessibilityRole="button"
              hitSlop={12}
              onPress={() => setSafetyOpen(true)}
            >
              <Ionicons color={theme.text} name="ellipsis-horizontal" size={24} />
            </Pressable>
          )
    });
  }, [navigation, profile?.firstName, firstName, isSelf, theme.text]);

  const act = async (action: SwipeAction) => {
    if (!profile || acting) return;
    setActing(true);
    try {
      markProfileResolved(profile.id);
      const result = await swipe(profile.id, action);
      if (result.matched && result.match && !result.alreadyProcessed) {
        announceMatch(result.match);
      }
      navigation.goBack();
    } catch (error) {
      toast.show(errorMessage(error), "error");
      setActing(false);
    }
  };

  const onScroll = (event: NativeSyntheticEvent<NativeScrollEvent>) =>
    setPage(Math.round(event.nativeEvent.contentOffset.x / width));

  if (query.isLoading) return <LoadingView />;
  if (query.isError || !profile) {
    return (
      <ErrorView
        message={errorMessage(query.error, "Ce profil n'est plus disponible.")}
        onRetry={() => void query.refetch()}
      />
    );
  }
  const distance = formatDistance(profile.distanceKm);

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      <ScrollView contentContainerStyle={styles.content}>
        <View>
          <FlatList
            data={profile.photos}
            horizontal
            keyExtractor={(p) => p.id}
            onScroll={onScroll}
            pagingEnabled
            renderItem={({ item }) => (
              <AuthImage
                accessibilityLabel={`Photo de ${profile.firstName}`}
                style={{ width, height: width * 1.25 }}
                url={item.url}
              />
            )}
            scrollEventThrottle={32}
            showsHorizontalScrollIndicator={false}
          />
          {profile.photos.length > 1 ? (
            <View style={styles.dots}>
              {profile.photos.map((p, index) => (
                <View
                  key={p.id}
                  style={[
                    styles.dot,
                    { backgroundColor: index === page ? "#FFFFFF" : "rgba(255,255,255,0.5)" }
                  ]}
                />
              ))}
            </View>
          ) : null}
        </View>

        <View style={styles.info}>
          {isSelf ? (
            <Text
              style={[styles.note, { color: theme.textMuted, backgroundColor: theme.surfaceMuted }]}
            >
              Voici votre profil tel que les autres le voient.
            </Text>
          ) : null}
          <Text style={[styles.name, { color: theme.text }]}>
            {profile.firstName}, {profile.age}
          </Text>
          {profile.city || distance ? (
            <View style={styles.location}>
              <Ionicons color={theme.textMuted} name="location-outline" size={16} />
              <Text style={[styles.locationText, { color: theme.textMuted }]}>
                {[profile.city, distance].filter(Boolean).join(" · ")}
              </Text>
            </View>
          ) : null}
          {profile.bio ? (
            <>
              <Text style={[styles.section, { color: theme.textMuted }]}>À propos</Text>
              <Text style={[styles.bio, { color: theme.text }]}>{profile.bio}</Text>
            </>
          ) : null}
          {profile.interests.length > 0 ? (
            <>
              <Text style={[styles.section, { color: theme.textMuted }]}>
                Centres d&apos;intérêt
              </Text>
              <View style={styles.chips}>
                {profile.interests.map((i) => (
                  <Chip key={i.id} label={i.label} />
                ))}
              </View>
            </>
          ) : null}
        </View>
      </ScrollView>

      {allowActions && !isSelf ? (
        <View
          style={[
            styles.actions,
            { backgroundColor: theme.background, borderTopColor: theme.border }
          ]}
        >
          <Pressable
            accessibilityLabel="Passer"
            accessibilityRole="button"
            disabled={acting}
            onPress={() => void act("pass")}
            style={[styles.round, { backgroundColor: theme.surface, borderColor: theme.border }]}
          >
            <Ionicons color={theme.textMuted} name="close" size={32} />
          </Pressable>
          <Pressable
            accessibilityLabel="J'aime"
            accessibilityRole="button"
            disabled={acting}
            onPress={() => void act("like")}
            style={[styles.round, { backgroundColor: theme.primary, borderColor: theme.primary }]}
          >
            <Ionicons color="#FFFFFF" name="heart" size={30} />
          </Pressable>
        </View>
      ) : null}

      <SafetySheet
        firstName={profile.firstName}
        onBlocked={() => {
          setSafetyOpen(false);
          navigation.goBack();
        }}
        onClose={() => setSafetyOpen(false)}
        userId={profile.id}
        visible={safetyOpen}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { paddingBottom: spacing.xxl },
  dots: {
    position: "absolute",
    bottom: spacing.md,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
  info: { padding: spacing.lg },
  note: {
    fontSize: fontSizes.sm,
    padding: spacing.md,
    borderRadius: 12,
    overflow: "hidden",
    marginBottom: spacing.lg
  },
  name: { fontSize: 30, fontWeight: "700", letterSpacing: -0.5 },
  location: { flexDirection: "row", alignItems: "center", marginTop: spacing.xs, gap: 4 },
  locationText: { fontSize: fontSizes.md },
  section: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.8,
    textTransform: "uppercase",
    marginTop: spacing.xl,
    marginBottom: spacing.sm
  },
  bio: { fontSize: fontSizes.md, lineHeight: 24 },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    paddingVertical: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  round: {
    width: 64,
    height: 64,
    borderRadius: 32,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  }
});
