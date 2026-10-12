import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useLayoutEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { datingApi } from "../api/dating";
import { ActionMenu } from "../components/ActionMenu";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ProfileDetails } from "../components/ProfileCard";
import { keys, useUserProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { radii, useTheme } from "../theme";
import { EmptyState, ErrorState, LoadingState, Screen, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParamList, "UserProfile">;

export default function UserProfileScreen({ navigation, route }: Props) {
  const { userId } = route.params;
  const { colors } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const query = useUserProfile(userId);
  const [menu, setMenu] = useState(false);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: query.data?.firstName ?? "Profil",
      headerStyle: { backgroundColor: colors.background },
      headerTintColor: colors.text,
      headerShadowVisible: false,
      headerRight: () => (
        <Pressable accessibilityRole="button" accessibilityLabel="Options" onPress={() => setMenu(true)} hitSlop={10}>
          <Ionicons name="ellipsis-horizontal" size={24} color={colors.text} />
        </Pressable>
      )
    });
  }, [colors, navigation, query.data?.firstName]);

  const block = async () => {
    try {
      await datingApi.block(userId);
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.discover });
      toast.show("Profil bloqué", "success");
      navigation.popToTop();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError || !query.data) {
    const gone = query.error instanceof Error && /not found/i.test(query.error.message);
    return gone ? (
      <EmptyState
        icon="person-remove-outline"
        title="Profil indisponible"
        message="Cette personne n'est plus accessible."
        actionLabel="Retour"
        onAction={() => navigation.goBack()}
      />
    ) : (
      <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
    );
  }

  const card = query.data;
  return (
    <Screen scroll top={false}>
      <View style={styles.photo}>
        <PhotoCarousel photos={card.photos} name={card.firstName} />
      </View>
      <ProfileDetails card={card} />
      <ActionMenu
        visible={menu}
        title={card.firstName}
        onClose={() => setMenu(false)}
        actions={[
          { label: "Signaler", destructive: true, onPress: () => navigation.navigate("Report", { userId, name: card.firstName }) },
          { label: "Bloquer", destructive: true, onPress: () => void block() }
        ]}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({ photo: { height: 460, borderRadius: radii.xl, overflow: "hidden" } });
