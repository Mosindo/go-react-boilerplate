import React, { useLayoutEffect, useState } from "react";
import { Pressable, Text, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { discoverApi } from "../../api/endpoints";
import { CardDetails, CardPhoto } from "../../components/ProfileCardView";
import { SafetyMenu } from "../../components/SafetyMenu";
import { ErrorState, Loading, Screen } from "../../components/ui";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/theme";

type Props = NativeStackScreenProps<MainStackParamList, "UserProfile">;

export default function UserProfileScreen({ route, navigation }: Props) {
  const { colors } = useTheme();
  const { userId, matchId } = route.params;
  const [menu, setMenu] = useState(false);
  const query = useQuery({ queryKey: ["user", userId], queryFn: () => discoverApi.profile(userId) });
  const card = query.data;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: card?.firstName ?? "Profil",
      headerRight: () => (
        <Pressable accessibilityRole="button" accessibilityLabel="Plus d'options" onPress={() => setMenu(true)} hitSlop={12}>
          <Text style={{ color: colors.text, fontSize: 24, fontWeight: "800" }}>⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, card?.firstName, colors.text]);

  if (query.isLoading) return <Loading />;
  if (query.isError || !card)
    return <ErrorState message={apiErrorMessage(query.error, "Profil introuvable.")} onRetry={() => void query.refetch()} />;

  return (
    <Screen edges={["bottom"]} padded={false}>
      <View style={{ height: 460, margin: 16, marginBottom: 0 }}>
        <CardPhoto card={card} />
      </View>
      <CardDetails card={card} />
      <SafetyMenu
        visible={menu}
        onClose={() => setMenu(false)}
        userId={card.userId}
        name={card.firstName}
        matchId={matchId}
        onGone={() => navigation.popToTop()}
      />
    </Screen>
  );
}
