package com.bysinbin.oewatcher.data

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import com.bysinbin.oewatcher.engine.BlacklistMatcher
import com.bysinbin.oewatcher.model.AliasItem
import com.bysinbin.oewatcher.model.MediaTrackInfo
import com.bysinbin.oewatcher.model.PersonItem
import com.bysinbin.oewatcher.model.SkipEvent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

object MediaRepository {
    private const val TAG = "MediaRepository"
    private const val PREFS_NAME = "oe_watcher_prefs"
    private const val KEY_AUTO_SKIP = "auto_skip"
    private const val KEY_TOTAL_SKIPS = "total_skips"
    private const val KEY_SERVER_URL = "server_url"
    private const val KEY_HISTORY = "history_json"
    private const val KEY_CUSTOM_PERSONS = "custom_persons_json"
    private const val KEY_CUSTOM_ALIASES = "custom_aliases_json"
    private const val KEY_CUSTOM_EXCLUDES = "custom_excludes_json"

    private var prefs: SharedPreferences? = null

    private val _currentTrack = MutableStateFlow<MediaTrackInfo?>(null)
    val currentTrack: StateFlow<MediaTrackInfo?> = _currentTrack.asStateFlow()

    private val _isServiceRunning = MutableStateFlow(false)
    val isServiceRunning: StateFlow<Boolean> = _isServiceRunning.asStateFlow()

    private val _isPermissionGranted = MutableStateFlow(false)
    val isPermissionGranted: StateFlow<Boolean> = _isPermissionGranted.asStateFlow()

    private val _autoSkipEnabled = MutableStateFlow(true)
    val autoSkipEnabled: StateFlow<Boolean> = _autoSkipEnabled.asStateFlow()

    private val _totalSkips = MutableStateFlow(0)
    val totalSkips: StateFlow<Int> = _totalSkips.asStateFlow()

    private val _historyList = MutableStateFlow<List<SkipEvent>>(emptyList())
    val historyList: StateFlow<List<SkipEvent>> = _historyList.asStateFlow()

    private val _blacklistCount = MutableStateFlow(574)
    val blacklistCount: StateFlow<Int> = _blacklistCount.asStateFlow()

    private val _personsList = MutableStateFlow<List<PersonItem>>(emptyList())
    val personsList: StateFlow<List<PersonItem>> = _personsList.asStateFlow()

    private val _customPersonsList = MutableStateFlow<List<PersonItem>>(emptyList())
    val customPersonsList: StateFlow<List<PersonItem>> = _customPersonsList.asStateFlow()

    private val _aliasesList = MutableStateFlow<List<AliasItem>>(emptyList())
    val aliasesList: StateFlow<List<AliasItem>> = _aliasesList.asStateFlow()

    private val _excludeList = MutableStateFlow<List<String>>(emptyList())
    val excludeList: StateFlow<List<String>> = _excludeList.asStateFlow()

    private val _serverUrl = MutableStateFlow("http://192.168.1.100:8080")
    val serverUrl: StateFlow<String> = _serverUrl.asStateFlow()

    // Control events sent from UI to Service
    private val _playerActions = MutableSharedFlow<String>(extraBufferCapacity = 5)
    val playerActions: SharedFlow<String> = _playerActions.asSharedFlow()

    fun init(context: Context) {
        prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val savedAutoSkip = prefs?.getBoolean(KEY_AUTO_SKIP, true) ?: true
        val savedTotalSkips = prefs?.getInt(KEY_TOTAL_SKIPS, 0) ?: 0
        val savedServerUrl = prefs?.getString(KEY_SERVER_URL, "http://192.168.1.100:8080") ?: "http://192.168.1.100:8080"

        _autoSkipEnabled.value = savedAutoSkip
        _totalSkips.value = savedTotalSkips
        _serverUrl.value = savedServerUrl

        // 1. Load base offline blacklist from assets
        BlacklistMatcher.loadFromAssets(context)

        // 2. Load persistent Custom Persons
        loadCustomPersons()

        // 3. Load persistent Aliases
        loadCustomAliases()

        // 4. Load persistent Excludes
        loadCustomExcludes()

        // 5. Load persistent History
        loadHistory()

        // 6. Refresh UI flows
        refreshFlows()
    }

    private fun refreshFlows() {
        _blacklistCount.value = BlacklistMatcher.totalCount
        _personsList.value = BlacklistMatcher.getAllPersons()
        _customPersonsList.value = BlacklistMatcher.getCustomPersons()

        val allAliases = BlacklistMatcher.getAllAliases()
        _aliasesList.value = allAliases.map { AliasItem(alias = it.key, target = it.value) }

        _excludeList.value = BlacklistMatcher.getAllExcludes()
    }

    // ==========================================
    // PERSISTENCE HELPERS
    // ==========================================
    private fun loadCustomPersons() {
        try {
            val jsonStr = prefs?.getString(KEY_CUSTOM_PERSONS, null) ?: return
            val arr = JSONArray(jsonStr)
            val list = mutableListOf<PersonItem>()
            for (i in 0 until arr.length()) {
                val obj = arr.getJSONObject(i)
                val name = obj.optString("name")
                if (name.isBlank()) continue
                val tags = mutableListOf<String>()
                val tagsArr = obj.optJSONArray("tags")
                if (tagsArr != null) {
                    for (j in 0 until tagsArr.length()) tags.add(tagsArr.getString(j))
                }
                list.add(PersonItem(name = name, tags = tags))
            }
            BlacklistMatcher.setCustomPersons(list)
        } catch (e: Exception) {
            Log.e(TAG, "Error loading custom persons: ${e.message}")
        }
    }

    private fun saveCustomPersons() {
        try {
            val list = BlacklistMatcher.getCustomPersons()
            val arr = JSONArray()
            for (item in list) {
                val obj = JSONObject()
                obj.put("name", item.name)
                val tagsArr = JSONArray()
                item.tags.forEach { tagsArr.put(it) }
                obj.put("tags", tagsArr)
                arr.put(obj)
            }
            prefs?.edit()?.putString(KEY_CUSTOM_PERSONS, arr.toString())?.apply()
        } catch (e: Exception) {
            Log.e(TAG, "Error saving custom persons: ${e.message}")
        }
    }

    private fun loadCustomAliases() {
        try {
            val jsonStr = prefs?.getString(KEY_CUSTOM_ALIASES, null) ?: return
            val obj = JSONObject(jsonStr)
            val map = mutableMapOf<String, String>()
            val keys = obj.keys()
            while (keys.hasNext()) {
                val k = keys.next()
                map[k] = obj.getString(k)
            }
            BlacklistMatcher.setCustomAliases(map)
        } catch (e: Exception) {
            Log.e(TAG, "Error loading custom aliases: ${e.message}")
        }
    }

    private fun saveCustomAliases() {
        try {
            val map = BlacklistMatcher.getCustomAliases()
            val obj = JSONObject()
            for ((k, v) in map) {
                obj.put(k, v)
            }
            prefs?.edit()?.putString(KEY_CUSTOM_ALIASES, obj.toString())?.apply()
        } catch (e: Exception) {
            Log.e(TAG, "Error saving custom aliases: ${e.message}")
        }
    }

    private fun loadCustomExcludes() {
        try {
            val jsonStr = prefs?.getString(KEY_CUSTOM_EXCLUDES, null) ?: return
            val arr = JSONArray(jsonStr)
            val list = mutableListOf<String>()
            for (i in 0 until arr.length()) {
                val s = arr.getString(i)
                if (s.isNotBlank()) list.add(s)
            }
            BlacklistMatcher.setCustomExcludes(list)
        } catch (e: Exception) {
            Log.e(TAG, "Error loading custom excludes: ${e.message}")
        }
    }

    private fun saveCustomExcludes() {
        try {
            val list = BlacklistMatcher.getCustomExcludes()
            val arr = JSONArray()
            for (item in list) arr.put(item)
            prefs?.edit()?.putString(KEY_CUSTOM_EXCLUDES, arr.toString())?.apply()
        } catch (e: Exception) {
            Log.e(TAG, "Error saving custom excludes: ${e.message}")
        }
    }

    private fun loadHistory() {
        try {
            val jsonStr = prefs?.getString(KEY_HISTORY, null) ?: return
            val arr = JSONArray(jsonStr)
            val list = mutableListOf<SkipEvent>()
            for (i in 0 until arr.length()) {
                val obj = arr.getJSONObject(i)
                list.add(
                    SkipEvent(
                        id = obj.optString("id"),
                        app = obj.optString("app"),
                        artist = obj.optString("artist"),
                        title = obj.optString("title"),
                        matchedPerson = obj.optString("matchedPerson"),
                        timestamp = obj.optString("timestamp"),
                        success = obj.optBoolean("success", true)
                    )
                )
            }
            _historyList.value = list
        } catch (e: Exception) {
            Log.e(TAG, "Error loading history: ${e.message}")
        }
    }

    private fun saveHistory() {
        try {
            val arr = JSONArray()
            for (item in _historyList.value.take(100)) {
                val obj = JSONObject().apply {
                    put("id", item.id)
                    put("app", item.app)
                    put("artist", item.artist)
                    put("title", item.title)
                    put("matchedPerson", item.matchedPerson)
                    put("timestamp", item.timestamp)
                    put("success", item.success)
                }
                arr.put(obj)
            }
            prefs?.edit()?.putString(KEY_HISTORY, arr.toString())?.apply()
        } catch (e: Exception) {
            Log.e(TAG, "Error saving history: ${e.message}")
        }
    }

    // ==========================================
    // PUBLIC CRUD API
    // ==========================================
    fun updateCurrentTrack(track: MediaTrackInfo?) {
        _currentTrack.value = track
    }

    fun setServiceRunning(running: Boolean) {
        _isServiceRunning.value = running
    }

    fun setPermissionGranted(granted: Boolean) {
        _isPermissionGranted.value = granted
    }

    fun setAutoSkip(enabled: Boolean) {
        _autoSkipEnabled.value = enabled
        prefs?.edit()?.putBoolean(KEY_AUTO_SKIP, enabled)?.apply()
    }

    fun setServerUrl(url: String) {
        _serverUrl.value = url.trim()
        prefs?.edit()?.putString(KEY_SERVER_URL, url.trim())?.apply()
    }

    fun recordSkip(event: SkipEvent) {
        val updatedList = mutableListOf(event).apply {
            addAll(_historyList.value.take(99)) // Keep up to 100 recent items
        }
        _historyList.value = updatedList
        saveHistory()

        val newTotal = _totalSkips.value + 1
        _totalSkips.value = newTotal
        prefs?.edit()?.putInt(KEY_TOTAL_SKIPS, newTotal)?.apply()
    }

    fun clearHistory() {
        _historyList.value = emptyList()
        saveHistory()
    }

    // Custom Person CRUD
    fun addCustomPerson(name: String, tag: String) {
        val tags = if (tag.isNotBlank()) listOf(tag) else listOf("Özel")
        BlacklistMatcher.addCustomPerson(name, tags)
        saveCustomPersons()
        refreshFlows()
    }

    fun removeCustomPerson(name: String) {
        BlacklistMatcher.removeCustomPerson(name)
        saveCustomPersons()
        refreshFlows()
    }

    // Alias CRUD
    fun addAlias(alias: String, target: String) {
        BlacklistMatcher.addCustomAlias(alias, target)
        saveCustomAliases()
        refreshFlows()
    }

    fun removeAlias(alias: String) {
        BlacklistMatcher.removeCustomAlias(alias)
        saveCustomAliases()
        refreshFlows()
    }

    // Exclude CRUD
    fun addExclude(pattern: String) {
        BlacklistMatcher.addCustomExclude(pattern)
        saveCustomExcludes()
        refreshFlows()
    }

    fun removeExclude(pattern: String) {
        BlacklistMatcher.removeCustomExclude(pattern)
        saveCustomExcludes()
        refreshFlows()
    }

    fun triggerSkipNext() {
        _playerActions.tryEmit("SKIP_NEXT")
    }

    fun triggerPlayPause() {
        _playerActions.tryEmit("PLAY_PAUSE")
    }

    fun syncFromRemoteServer(targetUrl: String, onComplete: (Boolean, String) -> Unit) {
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val cleanUrl = targetUrl.trim().removeSuffix("/")
                val finalUrl = if (cleanUrl.endsWith("/api/extension/sync") || cleanUrl.endsWith(".json")) {
                    cleanUrl
                } else {
                    "$cleanUrl/api/extension/sync"
                }

                val url = URL(finalUrl)
                val conn = (url.openConnection() as HttpURLConnection).apply {
                    connectTimeout = 6000
                    readTimeout = 8000
                    requestMethod = "GET"
                    setRequestProperty("Accept", "application/json")
                }

                val responseCode = conn.responseCode
                if (responseCode in 200..299) {
                    val reader = BufferedReader(InputStreamReader(conn.inputStream))
                    val response = reader.readText()
                    reader.close()

                    val root = JSONObject(response)
                    BlacklistMatcher.parseJsonData(root)

                    // Re-apply local custom rules over the synced base
                    loadCustomPersons()
                    loadCustomAliases()
                    loadCustomExcludes()

                    withContext(Dispatchers.Main) {
                        refreshFlows()
                        onComplete(true, "Başarıyla ${BlacklistMatcher.totalCount} kural senkronize edildi!")
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        onComplete(false, "Sunucu yanıt kodu: $responseCode")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Senkronizasyon hatası: ${e.message}")
                withContext(Dispatchers.Main) {
                    onComplete(false, "Bağlantı hatası: ${e.localizedMessage ?: "Sunucuya ulaşılamadı"}")
                }
            }
        }
    }
}
