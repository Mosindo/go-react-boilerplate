import React from "react";
import { FlatList, Image, Pressable, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { AppText, EmptyState, ErrorState, Loading, Screen } from "../../components/ui";
import { discoveryApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { mediaUrl } from "../../api/config";
import { radius, spacing, useTheme } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";

export default function LikesScreen() {
  const { colors } = useTheme();
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const q = useQuery({ queryKey: ["likes"], queryFn: discoveryApi.likesReceived });

  return (
    <Screen>
      <AppText variant="title" style={{ paddingVertical: spacing.sm }}>
        Ils vous ont liké
      </AppText>
      <AppText variant="muted" style={{ marginBottom: spacing.md }}>
        Gratuit et sans limite : likez en retour pour créer un match.
      </AppText>
      {q.isLoading ? (
        <Loading />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : (q.data ?? []).length === 0 ? (
        <EmptyState
          title="Pas encore de like"
          message="Quand quelqu'un vous likera, son profil apparaîtra ici."
        />
      ) : (
        <FlatList
          data={q.data}
          numColumns={2}
          keyExtractor={(c) => c.userId}
          columnWrapperStyle={{ gap: spacing.md }}
          contentContainerStyle={{ gap: spacing.md, paddingBottom: spacing.xl }}
          refreshing={q.isRefetching}
          onRefresh={() => void q.refetch()}
          renderItem={({ item }) => (
            <Pressable
              testID={`like-${item.firstName}`}
              accessibilityRole="button"
              accessibilityLabel={`${item.firstName}, ${item.age} ans`}
              style={{ flex: 1 }}
              onPress={() => nav.navigate("ProfileDetail", { userId: item.userId, card: item, canAct: true })}
            >
              <Image
                source={{ uri: mediaUrl(item.photos[0]?.url) }}
                style={{ aspectRatio: 3 / 4, borderRadius: radius.md, backgroundColor: colors.surfaceAlt }}
              />
              <View
                style={{
                  position: "absolute",
                  left: 0,
                  right: 0,
                  bottom: 0,
                  padding: spacing.sm,
                  backgroundColor: "rgba(0,0,0,0.4)",
                  borderBottomLeftRadius: radius.md,
                  borderBottomRightRadius: radius.md,
                }}
              >
                <AppText variant="small" style={{ color: "#fff", fontWeight: "700" }}>
                  {item.firstName}, {item.age}
                </AppText>
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}
