package com.bysinbin.oewatcher.ui.main

import android.content.Context
import android.content.Intent
import android.provider.Settings
import android.widget.Toast
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.bysinbin.oewatcher.data.MediaRepository
import com.bysinbin.oewatcher.engine.BlacklistMatcher
import com.bysinbin.oewatcher.model.AliasItem
import com.bysinbin.oewatcher.model.ExcludeItem
import com.bysinbin.oewatcher.model.MediaTrackInfo
import com.bysinbin.oewatcher.model.PersonItem
import com.bysinbin.oewatcher.model.SkipEvent
import com.bysinbin.oewatcher.theme.AmberWarning
import com.bysinbin.oewatcher.theme.CrimsonLight
import com.bysinbin.oewatcher.theme.CrimsonPrimary
import com.bysinbin.oewatcher.theme.EmeraldSuccess
import com.bysinbin.oewatcher.theme.SkyInfo
import com.bysinbin.oewatcher.theme.SlateBackground
import com.bysinbin.oewatcher.theme.SlateBorder
import com.bysinbin.oewatcher.theme.SlateCard
import com.bysinbin.oewatcher.theme.SlateSurface
import com.bysinbin.oewatcher.theme.TextMuted
import com.bysinbin.oewatcher.theme.TextPrimary
import com.bysinbin.oewatcher.theme.TextSecondary
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun MainScreen() {
    val context = LocalContext.current
    val isPermissionGrantedRepo by MediaRepository.isPermissionGranted.collectAsStateWithLifecycle()
    val hasPermission = isPermissionGrantedRepo || checkNotificationPermission(context)
    var selectedTab by remember { mutableIntStateOf(0) }

    val currentTrack by MediaRepository.currentTrack.collectAsStateWithLifecycle()
    val isServiceRunning by MediaRepository.isServiceRunning.collectAsStateWithLifecycle()
    val autoSkipEnabled by MediaRepository.autoSkipEnabled.collectAsStateWithLifecycle()
    val totalSkips by MediaRepository.totalSkips.collectAsStateWithLifecycle()
    val blacklistCount by MediaRepository.blacklistCount.collectAsStateWithLifecycle()
    val historyList by MediaRepository.historyList.collectAsStateWithLifecycle()
    val personsList by MediaRepository.personsList.collectAsStateWithLifecycle()
    val customPersonsList by MediaRepository.customPersonsList.collectAsStateWithLifecycle()
    val aliasesList by MediaRepository.aliasesList.collectAsStateWithLifecycle()
    val excludeList by MediaRepository.excludeList.collectAsStateWithLifecycle()
    val serverUrl by MediaRepository.serverUrl.collectAsStateWithLifecycle()

    Scaffold(
        modifier = Modifier
            .fillMaxSize()
            .statusBarsPadding()
            .navigationBarsPadding(),
        containerColor = SlateBackground,
        bottomBar = {
            NavigationBar(
                containerColor = SlateCard,
                tonalElevation = 8.dp,
                modifier = Modifier
                    .clip(RoundedCornerShape(topStart = 20.dp, topEnd = 20.dp))
                    .border(1.dp, SlateBorder, RoundedCornerShape(topStart = 20.dp, topEnd = 20.dp))
            ) {
                val navItems = listOf(
                    Triple(0, "🛡️", "Kalkan"),
                    Triple(1, "📋", "Kurallar"),
                    Triple(2, "📜", "Geçmiş"),
                    Triple(3, "⚙️", "Ayarlar")
                )
                navItems.forEach { (index, icon, label) ->
                    val isSelected = selectedTab == index
                    NavigationBarItem(
                        selected = isSelected,
                        onClick = { selectedTab = index },
                        icon = { Text(text = icon, fontSize = 18.sp) },
                        label = {
                            Text(
                                text = label,
                                fontSize = 11.sp,
                                fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Medium
                            )
                        },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = CrimsonPrimary,
                            selectedTextColor = CrimsonLight,
                            indicatorColor = CrimsonPrimary.copy(alpha = 0.2f),
                            unselectedIconColor = TextMuted,
                            unselectedTextColor = TextMuted
                        )
                    )
                }
            }
        }
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            when (selectedTab) {
                0 -> DashboardTab(
                    isServiceRunning = isServiceRunning,
                    hasPermission = hasPermission,
                    currentTrack = currentTrack,
                    autoSkipEnabled = autoSkipEnabled,
                    totalSkips = totalSkips,
                    blacklistCount = blacklistCount,
                    historyList = historyList,
                    onOpenPermission = {
                        try {
                            context.startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS))
                        } catch (_: Exception) {}
                    }
                )
                1 -> RulesAndListsTab(
                    persons = personsList,
                    customPersons = customPersonsList,
                    aliases = aliasesList,
                    excludeList = excludeList,
                    onAddCustomPerson = { name, tag -> MediaRepository.addCustomPerson(name, tag) },
                    onRemoveCustomPerson = { name -> MediaRepository.removeCustomPerson(name) },
                    onAddAlias = { alias, target -> MediaRepository.addAlias(alias, target) },
                    onRemoveAlias = { alias -> MediaRepository.removeAlias(alias) },
                    onAddExclude = { pattern -> MediaRepository.addExclude(pattern) },
                    onRemoveExclude = { pattern -> MediaRepository.removeExclude(pattern) }
                )
                2 -> HistoryTab(
                    history = historyList,
                    onClear = { MediaRepository.clearHistory() }
                )
                3 -> SettingsTab(
                    serverUrl = serverUrl,
                    onSaveUrl = { MediaRepository.setServerUrl(it) },
                    onSync = { url, cb -> MediaRepository.syncFromRemoteServer(url, cb) },
                    onOpenSettings = {
                        try {
                            context.startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS))
                        } catch (_: Exception) {}
                    }
                )
            }
        }
    }
}

// ==========================================
// TAB 0: DASHBOARD
// ==========================================
@Composable
fun DashboardTab(
    isServiceRunning: Boolean,
    hasPermission: Boolean,
    currentTrack: MediaTrackInfo?,
    autoSkipEnabled: Boolean,
    totalSkips: Int,
    blacklistCount: Int,
    historyList: List<SkipEvent>,
    onOpenPermission: () -> Unit
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 16.dp),
        contentPadding = PaddingValues(top = 16.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // 1. Header
        item {
            HeaderSection(
                hasPermission = hasPermission,
                isServiceRunning = isServiceRunning,
                onBadgeClick = onOpenPermission
            )
        }

        // 2. Permission Warning Banner
        if (!hasPermission) {
            item {
                PermissionWarningCard(onEnableClick = onOpenPermission)
            }
        }

        // 3. Now Playing Card with Play/Skip Controls
        item {
            NowPlayingCard(track = currentTrack)
        }

        // 4. Auto-Skip Toggle Card
        item {
            AutoSkipCard(
                enabled = autoSkipEnabled,
                onToggle = { MediaRepository.setAutoSkip(it) }
            )
        }

        // 5. Stat Badges Grid
        item {
            StatGrid(
                totalSkips = totalSkips,
                blacklistCount = blacklistCount
            )
        }

        // 6. Recent Skip History Preview
        item {
            SectionHeader(
                title = "Son Atlanan Şarkılar",
                subtitle = "${historyList.size} kayıt"
            )
        }

        if (historyList.isEmpty()) {
            item {
                EmptyStateCard(message = "Henüz atlanan şarkı kaydı yok.\nKara listeden bir şarkı çaldığında burada görünecektir.")
            }
        } else {
            items(historyList.take(6)) { event ->
                SkipHistoryItemCard(event = event)
            }
        }
    }
}

// ==========================================
// TAB 1: RULES & LISTS (KURALLAR & LİSTELER)
// ==========================================
@Composable
fun RulesAndListsTab(
    persons: List<PersonItem>,
    customPersons: List<PersonItem>,
    aliases: List<AliasItem>,
    excludeList: List<String>,
    onAddCustomPerson: (String, String) -> Unit,
    onRemoveCustomPerson: (String) -> Unit,
    onAddAlias: (String, String) -> Unit,
    onRemoveAlias: (String) -> Unit,
    onAddExclude: (String) -> Unit,
    onRemoveExclude: (String) -> Unit
) {
    var selectedSubTab by remember { mutableIntStateOf(0) }
    val context = LocalContext.current

    var showAddPersonDialog by remember { mutableStateOf(false) }
    var showAddAliasDialog by remember { mutableStateOf(false) }
    var aliasInitialTarget by remember { mutableStateOf("") }
    var showAddExcludeDialog by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 16.dp)
            .padding(top = 16.dp)
    ) {
        // Top Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "Kurallar & Filtreler",
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary
                )
                Text(
                    text = "Kara liste, muafiyet ve mahlas yönetimi",
                    fontSize = 12.sp,
                    color = TextSecondary
                )
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        // Sub Tab Selector (Pill selector)
        val subTabs = listOf(
            "Genel (${persons.size})",
            "Özel (${customPersons.size})",
            "Muaf (${excludeList.size})",
            "Alias (${aliases.size})"
        )

        LazyRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            contentPadding = PaddingValues(vertical = 2.dp)
        ) {
            items(subTabs.indices.toList()) { index ->
                val isSelected = selectedSubTab == index
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(12.dp))
                        .background(if (isSelected) CrimsonPrimary else SlateCard)
                        .border(
                            1.dp,
                            if (isSelected) CrimsonLight else SlateBorder,
                            RoundedCornerShape(12.dp)
                        )
                        .clickable { selectedSubTab = index }
                        .padding(horizontal = 14.dp, vertical = 8.dp)
                ) {
                    Text(
                        text = subTabs[index],
                        fontSize = 12.sp,
                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Medium,
                        color = if (isSelected) Color.White else TextSecondary
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        // Content of Sub-Tab
        Box(modifier = Modifier.weight(1f)) {
            when (selectedSubTab) {
                0 -> GenelListSubTab(
                    persons = persons,
                    onOpenAddAlias = { targetName ->
                        aliasInitialTarget = targetName
                        showAddAliasDialog = true
                    },
                    onAddExclude = { name ->
                        onAddExclude(name)
                        Toast.makeText(context, "$name muaf listesine eklendi", Toast.LENGTH_SHORT).show()
                    }
                )
                1 -> CustomPersonsSubTab(
                    customPersons = customPersons,
                    onOpenAdd = { showAddPersonDialog = true },
                    onDelete = onRemoveCustomPerson
                )
                2 -> ExcludeListSubTab(
                    excludeList = excludeList,
                    onOpenAdd = { showAddExcludeDialog = true },
                    onDelete = onRemoveExclude
                )
                3 -> AliasListSubTab(
                    aliases = aliases,
                    onOpenAdd = {
                        aliasInitialTarget = ""
                        showAddAliasDialog = true
                    },
                    onDelete = onRemoveAlias
                )
            }
        }
    }

    if (showAddPersonDialog) {
        AddCustomPersonDialog(
            onDismiss = { showAddPersonDialog = false },
            onConfirm = { name, tag ->
                onAddCustomPerson(name, tag)
                showAddPersonDialog = false
            }
        )
    }

    if (showAddAliasDialog) {
        AddAliasDialog(
            initialTarget = aliasInitialTarget,
            onDismiss = { showAddAliasDialog = false },
            onConfirm = { alias, target ->
                onAddAlias(alias, target)
                showAddAliasDialog = false
            }
        )
    }

    if (showAddExcludeDialog) {
        AddExcludeDialog(
            onDismiss = { showAddExcludeDialog = false },
            onConfirm = { pattern ->
                onAddExclude(pattern)
                showAddExcludeDialog = false
            }
        )
    }
}

@Composable
fun GenelListSubTab(
    persons: List<PersonItem>,
    onOpenAddAlias: (String) -> Unit,
    onAddExclude: (String) -> Unit
) {
    var searchQuery by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf("Tümü") }

    val categories = listOf("Tümü", "Şarkıcı", "Oyuncu", "Yazar", "Gazeteci", "Influencer", "Spor", "Marka", "Aktivist", "Din Adamı", "Diğer")

    val filteredList = remember(searchQuery, selectedCategory, persons) {
        val q = BlacklistMatcher.foldTurkish(searchQuery)
        val cat = BlacklistMatcher.foldTurkish(selectedCategory)
        persons.filter { person ->
            val matchesQuery = q.isBlank() ||
                    BlacklistMatcher.foldTurkish(person.name).contains(q) ||
                    person.tags.any { BlacklistMatcher.foldTurkish(it).contains(q) } ||
                    person.topics.any { BlacklistMatcher.foldTurkish(it).contains(q) }

            val matchesCategory = selectedCategory == "Tümü" ||
                    person.tags.any { BlacklistMatcher.foldTurkish(it) == cat }

            matchesQuery && matchesCategory
        }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        // Search Bar
        OutlinedTextField(
            value = searchQuery,
            onValueChange = { searchQuery = it },
            placeholder = { Text(text = "Sanatçı, kişi veya etiket ara...", color = TextMuted, fontSize = 13.sp) },
            trailingIcon = {
                if (searchQuery.isNotBlank()) {
                    Text(
                        text = "✕",
                        color = TextMuted,
                        fontSize = 14.sp,
                        modifier = Modifier
                            .clickable { searchQuery = "" }
                            .padding(8.dp)
                    )
                }
            },
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(12.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedContainerColor = SlateCard,
                unfocusedContainerColor = SlateCard,
                focusedBorderColor = CrimsonPrimary,
                unfocusedBorderColor = SlateBorder,
                focusedTextColor = TextPrimary,
                unfocusedTextColor = TextPrimary
            ),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(10.dp))

        // Category Filter Chips
        LazyRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            contentPadding = PaddingValues(vertical = 4.dp)
        ) {
            items(categories) { cat ->
                val isSelected = selectedCategory == cat
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(20.dp))
                        .background(if (isSelected) CrimsonPrimary else SlateCard)
                        .border(
                            1.dp,
                            if (isSelected) CrimsonLight else SlateBorder,
                            RoundedCornerShape(20.dp)
                        )
                        .clickable { selectedCategory = cat }
                        .padding(horizontal = 12.dp, vertical = 6.dp)
                ) {
                    Text(
                        text = cat,
                        fontSize = 11.sp,
                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                        color = if (isSelected) Color.White else TextSecondary
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "${filteredList.size} kişi eşleşti",
            fontSize = 11.sp,
            color = TextMuted
        )

        Spacer(modifier = Modifier.height(6.dp))

        // List
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (filteredList.isEmpty()) {
                item {
                    EmptyStateCard(message = "Aramanız veya seçili kategoriyle eşleşen kişi bulunamadı.")
                }
            } else {
                items(filteredList, key = { it.name }) { person ->
                    GenelPersonCard(
                        person = person,
                        onAddAlias = { onOpenAddAlias(person.name) },
                        onAddExclude = { onAddExclude(person.name) }
                    )
                }
            }
        }
    }
}

@Composable
fun GenelPersonCard(
    person: PersonItem,
    onAddAlias: () -> Unit,
    onAddExclude: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = SlateCard),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 12.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = person.name,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary,
                    modifier = Modifier.weight(1f)
                )

                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    // + Alias button
                    Box(
                        modifier = Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(SlateSurface)
                            .border(1.dp, SlateBorder, RoundedCornerShape(8.dp))
                            .clickable { onAddAlias() }
                            .padding(horizontal = 8.dp, vertical = 4.dp)
                    ) {
                        Text(text = "+ Alias", fontSize = 11.sp, color = SkyInfo, fontWeight = FontWeight.Bold)
                    }

                    // Muaf Tut button
                    Box(
                        modifier = Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(EmeraldSuccess.copy(alpha = 0.15f))
                            .border(1.dp, EmeraldSuccess.copy(alpha = 0.4f), RoundedCornerShape(8.dp))
                            .clickable { onAddExclude() }
                            .padding(horizontal = 8.dp, vertical = 4.dp)
                    ) {
                        Text(text = "🛡️ Muaf", fontSize = 11.sp, color = EmeraldSuccess, fontWeight = FontWeight.Bold)
                    }
                }
            }

            if (person.tags.isNotEmpty() || person.topics.isNotEmpty()) {
                Spacer(modifier = Modifier.height(4.dp))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    person.tags.forEach { tag ->
                        Box(
                            modifier = Modifier
                                .clip(RoundedCornerShape(6.dp))
                                .background(CrimsonPrimary.copy(alpha = 0.15f))
                                .padding(horizontal = 6.dp, vertical = 2.dp)
                        ) {
                            Text(text = tag, fontSize = 10.sp, color = CrimsonLight)
                        }
                    }
                    if (person.topics.isNotEmpty()) {
                        Text(
                            text = person.topics.joinToString(" • "),
                            fontSize = 11.sp,
                            color = TextMuted,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun CustomPersonsSubTab(
    customPersons: List<PersonItem>,
    onOpenAdd: () -> Unit,
    onDelete: (String) -> Unit
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(text = "Özel Kara Liste", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = TextPrimary)
                Text(text = "${customPersons.size} özel eklenmiş kişi", fontSize = 12.sp, color = TextSecondary)
            }
            Button(
                onClick = onOpenAdd,
                colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary),
                shape = RoundedCornerShape(10.dp),
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
            ) {
                Text("➕ Kişi Ekle", fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (customPersons.isEmpty()) {
                item {
                    EmptyStateCard(message = "Henüz özel eklenmiş kişi yok.\nYukarıdaki '➕ Kişi Ekle' butonuyla kendi listenize yeni sanatçı veya kişiler ekleyebilirsiniz.")
                }
            } else {
                items(customPersons, key = { it.name }) { item ->
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(12.dp),
                        colors = CardDefaults.cardColors(containerColor = SlateCard),
                        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 14.dp, vertical = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(text = item.name, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = TextPrimary)
                                if (item.tags.isNotEmpty()) {
                                    Spacer(modifier = Modifier.height(3.dp))
                                    Text(text = item.tags.joinToString(" • "), fontSize = 11.sp, color = CrimsonLight)
                                }
                            }
                            Box(
                                modifier = Modifier
                                    .size(34.dp)
                                    .clip(CircleShape)
                                    .background(Color.Red.copy(alpha = 0.1f))
                                    .clickable { onDelete(item.name) },
                                contentAlignment = Alignment.Center
                            ) {
                                Text(text = "🗑️", fontSize = 14.sp)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun ExcludeListSubTab(
    excludeList: List<String>,
    onOpenAdd: () -> Unit,
    onDelete: (String) -> Unit
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(text = "Muaf Listesi (Beyaz Liste)", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = TextPrimary)
                Text(text = "${excludeList.size} kural muaf tutuluyor", fontSize = 12.sp, color = TextSecondary)
            }
            Button(
                onClick = onOpenAdd,
                colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess),
                shape = RoundedCornerShape(10.dp),
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
            ) {
                Text("➕ Muaf Ekle", fontSize = 12.sp, fontWeight = FontWeight.Bold, color = Color.Black)
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Bu listedeki sanatçı ve şarkılar kara listede olsa bile asla engellenmez.",
            fontSize = 11.sp,
            color = TextMuted
        )

        Spacer(modifier = Modifier.height(10.dp))

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (excludeList.isEmpty()) {
                item {
                    EmptyStateCard(message = "Muaf listesi boş.\nAsla engellenmesini istemediğiniz sanatçıları buradan muaf tutabilirsiniz.")
                }
            } else {
                items(excludeList, key = { it }) { pattern ->
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(12.dp),
                        colors = CardDefaults.cardColors(containerColor = SlateCard),
                        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 14.dp, vertical = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Row(
                                modifier = Modifier.weight(1f),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(text = "🛡️", fontSize = 16.sp)
                                Spacer(modifier = Modifier.width(10.dp))
                                Text(
                                    text = pattern,
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.Medium,
                                    color = TextPrimary
                                )
                            }
                            Box(
                                modifier = Modifier
                                    .size(34.dp)
                                    .clip(CircleShape)
                                    .background(Color.Red.copy(alpha = 0.1f))
                                    .clickable { onDelete(pattern) },
                                contentAlignment = Alignment.Center
                            ) {
                                Text(text = "🗑️", fontSize = 14.sp)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun AliasListSubTab(
    aliases: List<AliasItem>,
    onOpenAdd: () -> Unit,
    onDelete: (String) -> Unit
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(text = "Mahlas & Sahne İsimleri", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = TextPrimary)
                Text(text = "${aliases.size} mahlas tanımlı", fontSize = 12.sp, color = TextSecondary)
            }
            Button(
                onClick = onOpenAdd,
                colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary),
                shape = RoundedCornerShape(10.dp),
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
            ) {
                Text("➕ Alias Ekle", fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Sanatçıların sahne adlarını ve kısaltmalarını gerçek adlarıyla eşleştirin.",
            fontSize = 11.sp,
            color = TextMuted
        )

        Spacer(modifier = Modifier.height(10.dp))

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (aliases.isEmpty()) {
                item {
                    EmptyStateCard(message = "Henüz mahlas tanımlanmadı.\nÖrn: 'Sansar' ➔ 'Sansar Salvo'")
                }
            } else {
                items(aliases, key = { it.alias }) { item ->
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(12.dp),
                        colors = CardDefaults.cardColors(containerColor = SlateCard),
                        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 14.dp, vertical = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Row(
                                modifier = Modifier.weight(1f),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Box(
                                    modifier = Modifier
                                        .clip(RoundedCornerShape(6.dp))
                                        .background(CrimsonPrimary.copy(alpha = 0.15f))
                                        .border(1.dp, CrimsonPrimary.copy(alpha = 0.3f), RoundedCornerShape(6.dp))
                                        .padding(horizontal = 8.dp, vertical = 3.dp)
                                ) {
                                    Text(text = item.alias, fontSize = 12.sp, fontWeight = FontWeight.Bold, color = CrimsonLight)
                                }
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(text = "➔", fontSize = 13.sp, color = TextMuted)
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(text = item.target, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = TextPrimary)
                            }
                            Box(
                                modifier = Modifier
                                    .size(34.dp)
                                    .clip(CircleShape)
                                    .background(Color.Red.copy(alpha = 0.1f))
                                    .clickable { onDelete(item.alias) },
                                contentAlignment = Alignment.Center
                            ) {
                                Text(text = "🗑️", fontSize = 14.sp)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun AddCustomPersonDialog(onDismiss: () -> Unit, onConfirm: (String, String) -> Unit) {
    var name by remember { mutableStateOf("") }
    var selectedTag by remember { mutableStateOf("Şarkıcı") }

    val presetTags = listOf("Şarkıcı", "Oyuncu", "Yazar", "Gazeteci", "Influencer", "Spor", "Marka", "Aktivist", "Diğer")

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = SlateSurface,
        title = {
            Text(text = "Özel Kara Liste Kişisi Ekle", color = TextPrimary, fontWeight = FontWeight.Bold)
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Kişi veya Sanatçı Adı", color = TextSecondary) },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedTextColor = TextPrimary,
                        unfocusedTextColor = TextPrimary,
                        focusedBorderColor = CrimsonPrimary,
                        unfocusedBorderColor = SlateBorder
                    ),
                    modifier = Modifier.fillMaxWidth()
                )

                Text(text = "Kategori Seçin:", fontSize = 12.sp, color = TextSecondary)

                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    contentPadding = PaddingValues(vertical = 2.dp)
                ) {
                    items(presetTags) { tag ->
                        val isSelected = selectedTag == tag
                        Box(
                            modifier = Modifier
                                .clip(RoundedCornerShape(12.dp))
                                .background(if (isSelected) CrimsonPrimary else SlateCard)
                                .border(1.dp, if (isSelected) CrimsonLight else SlateBorder, RoundedCornerShape(12.dp))
                                .clickable { selectedTag = tag }
                                .padding(horizontal = 10.dp, vertical = 6.dp)
                        ) {
                            Text(
                                text = tag,
                                fontSize = 11.sp,
                                color = if (isSelected) Color.White else TextSecondary,
                                fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = { if (name.isNotBlank()) onConfirm(name.trim(), selectedTag) },
                colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary)
            ) {
                Text("Ekle")
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("İptal", color = TextSecondary)
            }
        }
    )
}

@Composable
fun AddAliasDialog(
    initialTarget: String = "",
    onDismiss: () -> Unit,
    onConfirm: (String, String) -> Unit
) {
    var alias by remember { mutableStateOf("") }
    var target by remember { mutableStateOf(initialTarget) }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = SlateSurface,
        title = {
            Text(text = "Yeni Mahlas (Alias) Ekle", color = TextPrimary, fontWeight = FontWeight.Bold)
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = alias,
                    onValueChange = { alias = it },
                    label = { Text("Mahlas / Sahne Adı (Örn: Blok)", color = TextSecondary) },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedTextColor = TextPrimary,
                        unfocusedTextColor = TextPrimary,
                        focusedBorderColor = CrimsonPrimary,
                        unfocusedBorderColor = SlateBorder
                    ),
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = target,
                    onValueChange = { target = it },
                    label = { Text("Gerçek Adı / Hedef Sanatçı", color = TextSecondary) },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedTextColor = TextPrimary,
                        unfocusedTextColor = TextPrimary,
                        focusedBorderColor = CrimsonPrimary,
                        unfocusedBorderColor = SlateBorder
                    ),
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    if (alias.isNotBlank() && target.isNotBlank()) {
                        onConfirm(alias.trim(), target.trim())
                    }
                },
                colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary)
            ) {
                Text("Kaydet")
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("İptal", color = TextSecondary)
            }
        }
    )
}

@Composable
fun AddExcludeDialog(onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var pattern by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = SlateSurface,
        title = {
            Text(text = "Muaf Listesine Ekle", color = TextPrimary, fontWeight = FontWeight.Bold)
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = "Bu listedeki sanatçı veya şarkılar asla engellenmez.",
                    fontSize = 12.sp,
                    color = TextSecondary
                )
                OutlinedTextField(
                    value = pattern,
                    onValueChange = { pattern = it },
                    label = { Text("Sanatçı, Şarkı veya Kelime", color = TextSecondary) },
                    placeholder = { Text("Örn: Cem Adrian", color = TextMuted) },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedTextColor = TextPrimary,
                        unfocusedTextColor = TextPrimary,
                        focusedBorderColor = EmeraldSuccess,
                        unfocusedBorderColor = SlateBorder
                    ),
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { if (pattern.isNotBlank()) onConfirm(pattern.trim()) },
                colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess)
            ) {
                Text("Muaf Tut", color = Color.Black, fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("İptal", color = TextSecondary)
            }
        }
    )
}

// ==========================================
// TAB 2: HISTORY
// ==========================================
@Composable
fun HistoryTab(history: List<SkipEvent>, onClear: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 16.dp)
            .padding(top = 16.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "Engelleme Geçmişi",
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary
                )
                Text(
                    text = "Son ${history.size} otomatik atlama kaydı",
                    fontSize = 12.sp,
                    color = TextSecondary
                )
            }
            if (history.isNotEmpty()) {
                Button(
                    onClick = onClear,
                    colors = ButtonDefaults.buttonColors(containerColor = SlateCard),
                    border = ButtonDefaults.outlinedButtonBorder.copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder)),
                    shape = RoundedCornerShape(10.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
                ) {
                    Text(text = "Temizle", fontSize = 12.sp, color = TextMuted)
                }
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            if (history.isEmpty()) {
                item {
                    EmptyStateCard(message = "Henüz atlanan şarkı kaydı bulunmuyor.")
                }
            } else {
                items(history) { event ->
                    SkipHistoryItemCard(event = event)
                }
            }
        }
    }
}

// ==========================================
// TAB 3: SETTINGS & PC SYNC
// ==========================================
@Composable
fun SettingsTab(
    serverUrl: String,
    onSaveUrl: (String) -> Unit,
    onSync: (String, (Boolean, String) -> Unit) -> Unit,
    onOpenSettings: () -> Unit
) {
    var urlText by remember { mutableStateOf(serverUrl) }
    var isSyncing by remember { mutableStateOf(false) }
    val context = LocalContext.current

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 16.dp),
        contentPadding = PaddingValues(top = 16.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        item {
            Text(
                text = "Ayarlar & Senkronizasyon",
                fontSize = 20.sp,
                fontWeight = FontWeight.Bold,
                color = TextPrimary
            )
            Text(
                text = "Bilgisayarınızla eşitleme ve servis ayarları",
                fontSize = 12.sp,
                color = TextSecondary
            )
        }

        // PC Sync Card
        item {
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = SlateCard),
                border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
            ) {
                Column(modifier = Modifier.padding(18.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(text = "💻", fontSize = 20.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Column {
                            Text(
                                text = "Bilgisayarla Eşitle (PC Sync)",
                                fontSize = 16.sp,
                                fontWeight = FontWeight.Bold,
                                color = TextPrimary
                            )
                            Text(
                                text = "Bilgisayardaki güncel listeyi telefona çekin",
                                fontSize = 12.sp,
                                color = TextSecondary
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    OutlinedTextField(
                        value = urlText,
                        onValueChange = {
                            urlText = it
                            onSaveUrl(it)
                        },
                        label = { Text("Bilgisayar OE Watcher Adresi", color = TextSecondary) },
                        placeholder = { Text("http://192.168.1.X:8080", color = TextMuted) },
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(12.dp),
                        singleLine = true,
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedContainerColor = SlateSurface,
                            unfocusedContainerColor = SlateSurface,
                            focusedBorderColor = CrimsonPrimary,
                            unfocusedBorderColor = SlateBorder,
                            focusedTextColor = TextPrimary,
                            unfocusedTextColor = TextPrimary
                        )
                    )

                    Spacer(modifier = Modifier.height(12.dp))

                    Button(
                        onClick = {
                            isSyncing = true
                            onSync(urlText) { success, msg ->
                                isSyncing = false
                                Toast.makeText(context, msg, Toast.LENGTH_LONG).show()
                            }
                        },
                        enabled = !isSyncing,
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary),
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        if (isSyncing) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(18.dp),
                                color = Color.White,
                                strokeWidth = 2.dp
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text("Eşitleniyor...")
                        } else {
                            Text("🔄 Şimdi Senkronize Et", fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }
        }

        // Xiaomi & Battery Tips Card
        item {
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = SlateCard),
                border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
            ) {
                Column(modifier = Modifier.padding(18.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(text = "⚡", fontSize = 20.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Text(
                            text = "Xiaomi / Arka Plan Güvencesi",
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            color = TextPrimary
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "• Servis bildirim çubuğunda kalıcı olarak çalışır.\n" +
                                "• Xiaomi'de kesintisiz çalışması için Ayarlar > Uygulamalar > OE Watcher bölümünden 'Otomatik Başlatma'yı açık tutmanız önerilir.\n" +
                                "• Pil Tasarrufu ayarını 'Kısıtlama Yok' olarak ayarlayabilirsiniz.",
                        fontSize = 12.sp,
                        color = TextSecondary,
                        lineHeight = 18.sp
                    )
                    Spacer(modifier = Modifier.height(12.dp))
                    OutlinedButton(
                        onClick = onOpenSettings,
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(10.dp)
                    ) {
                        Text("Bildirim Erişimi Ayarlarını Aç", color = CrimsonLight, fontSize = 12.sp)
                    }
                }
            }
        }

        // App Info
        item {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 12.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Text(text = "OE Watcher Android Native", fontSize = 12.sp, fontWeight = FontWeight.Bold, color = TextMuted)
                Text(text = "Sürüm 1.0 • Offline-First + PC Sync", fontSize = 10.sp, color = TextMuted)
            }
        }
    }
}

// ==========================================
// SHARED COMPONENTS
// ==========================================
@Composable
fun HeaderSection(hasPermission: Boolean, isServiceRunning: Boolean, onBadgeClick: () -> Unit) {
    val isProtected = hasPermission
    val badgeText = if (hasPermission) "KORUMA AKTİF" else "İZİN GEREKLİ"
    val badgeColor = if (hasPermission) EmeraldSuccess else AmberWarning

    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(46.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(CrimsonPrimary.copy(alpha = 0.18f))
                    .border(1.dp, CrimsonPrimary.copy(alpha = 0.4f), RoundedCornerShape(12.dp)),
                contentAlignment = Alignment.Center
            ) {
                Text(text = "🛡️", fontSize = 22.sp)
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column {
                Text(
                    text = "OE WATCHER",
                    fontSize = 19.sp,
                    fontWeight = FontWeight.Black,
                    letterSpacing = 1.sp,
                    color = TextPrimary
                )
                Text(
                    text = "MEDYA & İÇERİK KALKANI",
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Bold,
                    letterSpacing = 0.8.sp,
                    color = CrimsonLight
                )
            }
        }

        // Service badge
        Row(
            modifier = Modifier
                .clip(CircleShape)
                .background(badgeColor.copy(alpha = 0.15f))
                .border(
                    1.dp,
                    badgeColor.copy(alpha = 0.4f),
                    CircleShape
                )
                .clickable { onBadgeClick() }
                .padding(horizontal = 10.dp, vertical = 5.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(7.dp)
                    .clip(CircleShape)
                    .background(badgeColor)
            )
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = badgeText,
                fontSize = 10.sp,
                fontWeight = FontWeight.Bold,
                color = badgeColor
            )
        }
    }
}

@Composable
fun NowPlayingCard(track: MediaTrackInfo?) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = SlateCard),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
    ) {
        Column(modifier = Modifier.padding(18.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(text = "🎵", fontSize = 16.sp)
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        text = "ŞU AN ÇALAN MEDYA",
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        letterSpacing = 0.8.sp,
                        color = SkyInfo
                    )
                }

                if (track != null) {
                    Box(
                        modifier = Modifier
                            .clip(RoundedCornerShape(6.dp))
                            .background(CrimsonPrimary.copy(alpha = 0.2f))
                            .padding(horizontal = 8.dp, vertical = 3.dp)
                    ) {
                        Text(
                            text = track.app.uppercase(),
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Bold,
                            color = CrimsonLight
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(14.dp))

            if (track != null && (track.title.isNotBlank() || track.artist.isNotBlank())) {
                Text(
                    text = track.title.ifBlank { "İsimsiz Parça" },
                    fontSize = 17.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(modifier = Modifier.height(3.dp))
                Text(
                    text = track.artist.ifBlank { "Bilinmeyen Sanatçı" },
                    fontSize = 14.sp,
                    color = CrimsonLight,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                if (track.album.isNotBlank()) {
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = track.album,
                        fontSize = 11.sp,
                        color = TextMuted,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }

                Spacer(modifier = Modifier.height(14.dp))

                // Play / Skip Buttons
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Button(
                        onClick = { MediaRepository.triggerPlayPause() },
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.buttonColors(containerColor = SlateSurface),
                        border = ButtonDefaults.outlinedButtonBorder.copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder)),
                        shape = RoundedCornerShape(10.dp)
                    ) {
                        Text(text = "⏯️ Oynat / Durdur", fontSize = 12.sp, color = TextPrimary)
                    }

                    Button(
                        onClick = { MediaRepository.triggerSkipNext() },
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.buttonColors(containerColor = CrimsonPrimary.copy(alpha = 0.25f)),
                        border = ButtonDefaults.outlinedButtonBorder.copy(brush = androidx.compose.ui.graphics.SolidColor(CrimsonPrimary.copy(alpha = 0.5f))),
                        shape = RoundedCornerShape(10.dp)
                    ) {
                        Text(text = "⏭️ Sonraki Şarkı", fontSize = 12.sp, color = CrimsonLight, fontWeight = FontWeight.Bold)
                    }
                }
            } else {
                Text(
                    text = "Şu anda aktif çalan medya oturumu algılanmadı.",
                    fontSize = 14.sp,
                    color = TextMuted
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "Spotify veya YouTube Music'ten bir şarkı açtığınızda burada canlı olarak belirecektir.",
                    fontSize = 12.sp,
                    color = TextMuted.copy(alpha = 0.7f),
                    lineHeight = 16.sp
                )
            }
        }
    }
}

@Composable
fun AutoSkipCard(enabled: Boolean, onToggle: (Boolean) -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = SlateCard),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(18.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "Otomatik Şarkı Geçişi (Auto-Skip)",
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary
                )
                Spacer(modifier = Modifier.height(3.dp))
                Text(
                    text = "Kara listedeki sanatçılar çalmaya başladığında beklemeden bir sonraki parçaya atla.",
                    fontSize = 11.sp,
                    color = TextSecondary,
                    lineHeight = 16.sp
                )
            }
            Spacer(modifier = Modifier.width(12.dp))
            Switch(
                checked = enabled,
                onCheckedChange = onToggle,
                colors = SwitchDefaults.colors(
                    checkedThumbColor = Color.White,
                    checkedTrackColor = CrimsonPrimary,
                    uncheckedThumbColor = TextMuted,
                    uncheckedTrackColor = SlateSurface
                )
            )
        }
    }
}

@Composable
fun StatGrid(totalSkips: Int, blacklistCount: Int) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Card(
            modifier = Modifier.weight(1f),
            shape = RoundedCornerShape(14.dp),
            colors = CardDefaults.cardColors(containerColor = SlateCard),
            border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(text = "Toplam Atlanan", fontSize = 11.sp, color = TextSecondary)
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = totalSkips.toString(),
                    fontSize = 24.sp,
                    fontWeight = FontWeight.Black,
                    color = CrimsonPrimary
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(text = "Şarkı otomatik geçildi", fontSize = 10.sp, color = TextMuted)
            }
        }

        Card(
            modifier = Modifier.weight(1f),
            shape = RoundedCornerShape(14.dp),
            colors = CardDefaults.cardColors(containerColor = SlateCard),
            border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(text = "Kara Liste Veritabanı", fontSize = 11.sp, color = TextSecondary)
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = blacklistCount.toString(),
                    fontSize = 24.sp,
                    fontWeight = FontWeight.Black,
                    color = SkyInfo
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(text = "Kişi / Sanatçı aktif", fontSize = 10.sp, color = TextMuted)
            }
        }
    }
}

@Composable
fun SkipHistoryItemCard(event: SkipEvent) {
    val timeFormatted = remember(event.timestamp) {
        if (event.timestamp.contains(":")) {
            event.timestamp
        } else {
            try {
                SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date(event.timestamp.toLong()))
            } catch (_: Exception) {
                event.timestamp
            }
        }
    }

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = SlateCard),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(14.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(CrimsonPrimary.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Text(text = "⏩", fontSize = 16.sp)
            }

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = event.title,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = TextPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = "Kara listedeki: ${event.matchedPerson}",
                    fontSize = 11.sp,
                    color = CrimsonLight,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            Column(horizontalAlignment = Alignment.End) {
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(4.dp))
                        .background(SlateSurface)
                        .padding(horizontal = 6.dp, vertical = 2.dp)
                ) {
                    Text(
                        text = event.app,
                        fontSize = 9.sp,
                        fontWeight = FontWeight.Bold,
                        color = SkyInfo
                    )
                }
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = timeFormatted,
                    fontSize = 10.sp,
                    color = TextMuted
                )
            }
        }
    }
}

@Composable
fun SectionHeader(title: String, subtitle: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = title,
            fontSize = 16.sp,
            fontWeight = FontWeight.Bold,
            color = TextPrimary
        )
        Text(
            text = subtitle,
            fontSize = 11.sp,
            color = TextMuted
        )
    }
}

@Composable
fun EmptyStateCard(message: String) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = SlateCard.copy(alpha = 0.6f)),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(SlateBorder))
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = message,
                fontSize = 12.sp,
                color = TextMuted,
                textAlign = TextAlign.Center,
                lineHeight = 18.sp
            )
        }
    }
}

@Composable
fun PermissionWarningCard(onEnableClick: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = AmberWarning.copy(alpha = 0.12f)),
        border = CardDefaults.outlinedCardBorder().copy(brush = androidx.compose.ui.graphics.SolidColor(AmberWarning.copy(alpha = 0.5f)))
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(text = "⚠️", fontSize = 20.sp)
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = "Medya Dinleme İzni Gerekli",
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    color = AmberWarning
                )
            }
            Spacer(modifier = Modifier.height(6.dp))
            Text(
                text = "Spotify, YouTube Music ve diğer müzik çalarlardaki parçaları otomatik algılayıp atlayabilmek için Android Bildirim & Medya Erişimi izni vermeniz gerekir.",
                fontSize = 12.sp,
                color = TextSecondary,
                lineHeight = 17.sp
            )
            Spacer(modifier = Modifier.height(12.dp))
            Button(
                onClick = onEnableClick,
                colors = ButtonDefaults.buttonColors(containerColor = AmberWarning),
                shape = RoundedCornerShape(10.dp)
            ) {
                Text(
                    text = "İzin Ver",
                    color = Color.Black,
                    fontWeight = FontWeight.Bold,
                    fontSize = 13.sp
                )
            }
        }
    }
}

fun checkNotificationPermission(context: Context): Boolean {
    val pkgName = context.packageName
    val flat = Settings.Secure.getString(context.contentResolver, "enabled_notification_listeners")
    if (flat != null && flat.contains(pkgName)) return true

    return try {
        androidx.core.app.NotificationManagerCompat.getEnabledListenerPackages(context).contains(pkgName)
    } catch (_: Exception) {
        false
    }
}
