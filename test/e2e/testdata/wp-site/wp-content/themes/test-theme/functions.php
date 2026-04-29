<?php
add_action('init', 'fixture_boot');
function fixture_boot() {
    wp_remote_get('https://example.test/api');
    wp_redirect('/next');
}
