import React, { useCallback } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import { CompositeScreenProps, useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { likesReceived } from "../api/dating";
import type { PublicProfile } from "../api/types";
import { PhotoImage } from "../components/PhotoImage";
import { describeError } from "../lib/errors";
import { nameWithAge } from "../lib/format";
import type { RootStackParamList, TabParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Text, radii, spacing } from "../shared/ui";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Likes">, NativeStackScreenProps<RootStackParamList>>;

export default function LikesScreen({ navigation }: Props) {
  const likes = useQuery<PublicProfile[], Error>({ queryKey: ["likes"], queryFn: () => likesReceived(), staleTime: 5_000 });

  useFocusEffect(
    useCallback(() => {
      void likes.refetch();
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])
  );

  const items = likes.data ?? [];

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <View style={styles.padded}>
        <Header subtitle="Elles et ils ont déjà dit oui. Likez en retour pour créer un match." title="Ils vous aiment" />
      </View>
      {likes.isLoading ? (
        <LoadingView fullScreen label="Chargement…" />
      ) : likes.isError ? (
        <ErrorView message={describeError(likes.error)} onAction={() => void likes.refetch()} />
      ) : items.length === 0 ? (
        <EmptyView
          icon="heart-outline"
          message="Quand quelqu'un vous likera, vous le verrez ici : sans abonnement, sans flou."
          title="Personne pour l'instant"
        />
      ) : (
        <FlatList
          columnWrapperStyle={styles.row}
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(item) => item.userId}
          numColumns={2}
          refreshControl={<RefreshControl onRefresh={() => void likes.refetch()} refreshing={likes.isRefetching} />}
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`Voir le profil de ${item.firstName}`}
              accessibilityRole="button"
              onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId, context: "likes" })}
              style={styles.card}
            >
              <PhotoImage path={item.photos[0]?.url} style={StyleSheet.absoluteFill} />
              <View style={styles.label}>
                <Text numberOfLines={1} style={styles.white} weight="semibold">
                  {nameWithAge(item.firstName, item.age)}
                </Text>
              </View>
            </Pressable>
          )}
          testID="likes-list"
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  padded: { paddingHorizontal: spacing.xl, paddingTop: spacing.md },
  list: { paddingHorizontal: spacing.lg, paddingBottom: spacing.xl, gap: spacing.md },
  row: { gap: spacing.md },
  card: { flex: 1, aspectRatio: 3 / 4, borderRadius: radii.lg, overflow: "hidden" },
  label: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.md, backgroundColor: "rgba(20, 12, 8, 0.5)" },
  white: { color: "#FFFFFF" }
});
