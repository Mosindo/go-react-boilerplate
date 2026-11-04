import React, { useLayoutEffect, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { ScrollView, View } from "react-native";
import { Dimensions } from "react-native";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import { keys } from "../../hooks/queries";
import type { MainStackParams } from "../../navigation/types";
import { Button, ErrorState, Loader, Sheet, spacing, useColors } from "../../shared/ui";
import { PhotoCarousel } from "./PhotoCarousel";
import { ProfileBody } from "./ProfileSummary";
import { ReportSheet } from "./ReportSheet";
import { useBlock } from "./useSafetyActions";

export function ProfileDetailScreen({ route, navigation }: NativeStackScreenProps<MainStackParams, "ProfileDetail">) {
  const { userId } = route.params;
  const c = useColors();
  const query = useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.view(userId) });
  const [menu, setMenu] = useState(false);
  const [report, setReport] = useState(false);
  const name = query.data?.firstName ?? "ce profil";
  const askBlock = useBlock(userId, name, () => navigation.popToTop());

  useLayoutEffect(() => {
    navigation.setOptions({
      title: query.data?.firstName ?? "Profil",
      headerRight: () => <Button title="Options" variant="ghost" onPress={() => setMenu(true)} testID="profile-menu" />,
    });
  }, [navigation, query.data?.firstName]);

  if (query.isPending) return <Loader />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;

  const height = Math.round(Dimensions.get("window").width * 1.25);
  return (
    <View style={{ flex: 1, backgroundColor: c.background }}>
      <ScrollView contentContainerStyle={{ paddingBottom: spacing.xxl }}>
        <View style={{ height }}>
          <PhotoCarousel photos={query.data.photos} name={query.data.firstName} rounded={false} />
        </View>
        <View style={{ padding: spacing.lg }}>
          <ProfileBody profile={query.data} />
        </View>
      </ScrollView>
      <Sheet visible={menu} onClose={() => setMenu(false)} title={query.data.firstName}>
        <Button title="Signaler" variant="secondary" onPress={() => { setMenu(false); setReport(true); }} testID="menu-report" />
        <Button title="Bloquer" variant="danger" onPress={() => { setMenu(false); askBlock(); }} testID="menu-block" />
      </Sheet>
      <ReportSheet userId={userId} visible={report} onClose={() => setReport(false)} onReported={() => { setReport(false); navigation.popToTop(); }} />
    </View>
  );
}
